import copy
import json
import os
import time
import traceback
from typing import Dict, Any, List, Tuple

import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
from peft import LoraConfig, get_peft_model, PeftModel
from psycopg2.extras import Json
from scipy.stats import pearsonr
from sklearn.model_selection import StratifiedKFold, KFold

from app.config import Config
from app.db import get_db_connection
from app.preprocess import split_key_points, build_answer_windows
from app.scorer import EmbeddingEngine, score_single_submission


def safe_pearson(y_true: np.ndarray, y_pred: np.ndarray) -> float:
    if len(y_true) < 2:
        return 0.0
    if np.std(y_true) < 1e-8 or np.std(y_pred) < 1e-8:
        return 0.0
    r_val, _ = pearsonr(y_true, y_pred)
    if np.isnan(r_val):
        return 0.0
    return float(r_val)


def disable_bert_dropout(module: nn.Module):
    for m in module.modules():
        if isinstance(m, nn.Dropout):
            m.p = 0.0


class DifferentiableKeyMatcherHead(nn.Module):
    """
    Menghitung 8 fitur agregat berukuran tetap secara differentiable dari skor mentah poin kunci,
    kemudian memetakannya ke skor ternormalisasi [0, 1] melalui head kecil (tanpa vektor 768 dimensi).
    8 fitur agregat:
      0: weighted_avg_coverage
      1: min_coverage
      2: max_coverage
      3: prop_criteria_terpenuhi
      4: prop_criteria_tidak_ada
      5: raw_point_mean (rata-rata cosine mentah semua poin kunci tanpa clip)
      6: raw_point_min  (cosine mentah minimum semua poin kunci)
      7: prop_points_tercakup (proporsi poin kunci yang tercakup pada level poin)
    """

    def __init__(self, t_low: float = 0.45, t_high: float = 0.60, hidden_dim: int = 8):
        super().__init__()
        mid_init = 0.5 * (t_low + t_high)
        self.mid_cov = nn.Parameter(torch.tensor(float(mid_init), dtype=torch.float32))
        self.raw_slope_cov = nn.Parameter(torch.tensor(14.0, dtype=torch.float32))
        self.mid_pt = nn.Parameter(torch.tensor(float(t_high), dtype=torch.float32))
        self.raw_slope_pt = nn.Parameter(torch.tensor(16.0, dtype=torch.float32))

        if hidden_dim > 0:
            self.head = nn.Sequential(
                nn.Linear(8, hidden_dim),
                nn.GELU(),
                nn.Linear(hidden_dim, 1),
            )
            nn.init.normal_(self.head[0].weight, mean=0.0, std=0.15)
        else:
            self.head = nn.Linear(8, 1)
            nn.init.normal_(self.head.weight, mean=0.0, std=0.15)

    def compute_agg_features(
        self,
        raw_point_scores: torch.Tensor,
        crit_slices: List[Tuple[int, int]],
        crit_weights: torch.Tensor,
    ) -> torch.Tensor:
        if raw_point_scores.numel() == 0:
            return torch.tensor([0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0], dtype=torch.float32)

        slope_cov = F.softplus(self.raw_slope_cov) + 1.0
        slope_pt = F.softplus(self.raw_slope_pt) + 1.0

        rescaled_pts = torch.sigmoid(slope_cov * (raw_point_scores - self.mid_cov))

        crit_covs = []
        for start_i, end_i in crit_slices:
            if end_i > start_i:
                crit_covs.append(torch.mean(rescaled_pts[start_i:end_i]))
            else:
                crit_covs.append(torch.tensor(0.0, dtype=torch.float32))
        cov_tensor = torch.stack(crit_covs)

        w_norm = crit_weights / torch.clamp(torch.sum(crit_weights), min=1e-6)
        weighted_avg_cov = torch.sum(w_norm * cov_tensor)
        min_cov = torch.min(cov_tensor)
        max_cov = torch.max(cov_tensor)

        prop_crit_terpenuhi = torch.mean(torch.sigmoid(12.0 * (cov_tensor - 0.75)))
        prop_crit_tidak_ada = torch.mean(torch.sigmoid(12.0 * (0.25 - cov_tensor)))

        raw_pt_mean = torch.mean(raw_point_scores)
        raw_pt_min = torch.min(raw_point_scores)
        prop_pts_tercakup = torch.mean(torch.sigmoid(slope_pt * (raw_point_scores - self.mid_pt)))

        return torch.stack(
            [
                weighted_avg_cov,
                min_cov,
                max_cov,
                prop_crit_terpenuhi,
                prop_crit_tidak_ada,
                raw_pt_mean,
                raw_pt_min,
                prop_pts_tercakup,
            ]
        )

    def forward_from_point_scores(
        self,
        raw_point_scores: torch.Tensor,
        crit_slices: List[Tuple[int, int]],
        crit_weights: torch.Tensor,
    ) -> torch.Tensor:
        agg = self.compute_agg_features(raw_point_scores, crit_slices, crit_weights)
        out = self.head(agg).squeeze(-1)
        return torch.sigmoid(out)


class CustomKeyMatcherModel(nn.Module):
    def __init__(self, encoder: nn.Module, matcher_head: DifferentiableKeyMatcherHead):
        super().__init__()
        self.encoder = encoder
        self.matcher_head = matcher_head

    def encode_batch(self, input_ids: torch.Tensor, attention_mask: torch.Tensor) -> torch.Tensor:
        outputs = self.encoder(input_ids=input_ids, attention_mask=attention_mask)
        token_embeddings = outputs[0]
        mask = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
        sum_embeddings = torch.sum(token_embeddings * mask, dim=1)
        sum_mask = torch.clamp(mask.sum(dim=1), min=1e-9)
        mean_pooled = sum_embeddings / sum_mask
        return F.normalize(mean_pooled, p=2, dim=1)

    def forward_single(
        self,
        win_input_ids: torch.Tensor,
        win_attention_mask: torch.Tensor,
        kp_input_ids: torch.Tensor,
        kp_attention_mask: torch.Tensor,
        crit_slices: List[Tuple[int, int]],
        crit_weights: torch.Tensor,
    ) -> torch.Tensor:
        win_vecs = self.encode_batch(win_input_ids, win_attention_mask)
        kp_vecs = self.encode_batch(kp_input_ids, kp_attention_mask)
        sim_matrix = torch.matmul(kp_vecs, win_vecs.T)
        raw_max_scores = torch.max(sim_matrix, dim=1).values
        return self.matcher_head.forward_from_point_scores(raw_max_scores, crit_slices, crit_weights)


class ActiveCustomPredictor:
    _cache: Dict[str, "ActiveCustomPredictor"] = {}

    def __init__(self, storage_path: str, training_mode: str = "calibration"):
        self.storage_path = storage_path
        self.training_mode = training_mode
        base_engine = EmbeddingEngine.get_instance()
        self.tokenizer = base_engine.tokenizer

        head_path = os.path.join(storage_path, "matcher_head.pt")
        self.matcher_head = DifferentiableKeyMatcherHead(hidden_dim=8)
        self.matcher_head.load_state_dict(torch.load(head_path, map_location="cpu"))
        self.matcher_head.to("cpu")
        self.matcher_head.eval()

        adapter_dir = os.path.join(storage_path, "adapter")
        self.custom_net = None
        if training_mode == "lora" and os.path.isdir(adapter_dir):
            base_copy = copy.deepcopy(base_engine.model)
            peft_encoder = PeftModel.from_pretrained(base_copy, adapter_dir)
            peft_encoder.to("cpu")
            peft_encoder.eval()
            self.custom_net = CustomKeyMatcherModel(peft_encoder, self.matcher_head)
            self.custom_net.eval()

    @torch.no_grad()
    def predict_norm(
        self,
        answer_text: str,
        raw_point_scores: List[float],
        crit_slices: List[Tuple[int, int]],
        crit_weights: List[float],
        all_key_points: List[str] = None,
    ) -> float:
        weights_t = torch.tensor(crit_weights, dtype=torch.float32)
        if self.custom_net is not None and all_key_points:
            windows = build_answer_windows(answer_text, tokenizer=self.tokenizer, max_tokens=512)
            if not windows:
                windows = [""]
            win_enc = self.tokenizer(
                windows, padding=True, truncation=True, max_length=512, return_tensors="pt"
            )
            kp_enc = self.tokenizer(
                all_key_points, padding=True, truncation=True, max_length=512, return_tensors="pt"
            )
            pred = self.custom_net.forward_single(
                win_enc["input_ids"],
                win_enc["attention_mask"],
                kp_enc["input_ids"],
                kp_enc["attention_mask"],
                crit_slices,
                weights_t,
            )
            return float(pred.item())

        raw_t = torch.tensor(raw_point_scores, dtype=torch.float32)
        pred = self.matcher_head.forward_from_point_scores(raw_t, crit_slices, weights_t)
        return float(pred.item())


def load_active_custom_predictor(conn, custom_model_id: int):
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT v.id, v.storage_path, COALESCE(v.training_mode, 'calibration') AS training_mode
            FROM custom_models m
            JOIN custom_model_versions v ON v.id = m.active_version_id
            WHERE m.id = %s
            """,
            (custom_model_id,),
        )
        row = cur.fetchone()
    if not row:
        return None

    storage_path = row["storage_path"]
    training_mode = row["training_mode"]
    if not os.path.exists(os.path.join(storage_path, "matcher_head.pt")):
        return None

    cache_key = f"{custom_model_id}:{row['id']}:{training_mode}:{storage_path}"
    if cache_key not in ActiveCustomPredictor._cache:
        print(
            f"[CustomPredictor] Memuat model custom aktif ({training_mode}) dari {storage_path}",
            flush=True,
        )
        ActiveCustomPredictor._cache[cache_key] = ActiveCustomPredictor(storage_path, training_mode)
    return ActiveCustomPredictor._cache[cache_key]


def update_training_job_progress(
    conn,
    r,
    job_id: str,
    status: str,
    stage: str,
    progress: int,
    error_message: str = "",
    metadata: Dict[str, Any] = None,
):
    meta_json = metadata or {}
    with conn.cursor() as cur:
        cur.execute(
            """
            UPDATE jobs
            SET status = %s,
                stage = %s,
                progress = %s,
                error_message = %s,
                metadata = %s,
                updated_at = NOW()
            WHERE id = %s
            """,
            (status, stage, progress, error_message, Json(meta_json), job_id),
        )
    conn.commit()

    redis_state = {
        "id": job_id,
        "status": status,
        "stage": stage,
        "progress": progress,
        "error_message": error_message,
        "metadata": meta_json,
    }
    r.set(f"job:progress:{job_id}", json.dumps(redis_state), ex=3600)


def compute_range_summary(targets: np.ndarray, base_preds: np.ndarray, custom_preds: np.ndarray) -> Dict[str, Any]:
    """
    Menghitung rata-rata nilai out-of-fold CV pada 3 rentang target nilai guru (skala 0-100):
    - tinggi (>= 80): jawaban lengkap
    - sedang (40 - 79.99): jawaban sebagian
    - rendah (< 40): jawaban melenceng
    """
    ranges = {
        "tinggi": targets >= 80.0,
        "sedang": (targets >= 40.0) & (targets < 80.0),
        "rendah": targets < 40.0,
    }
    out = {}
    for key, mask in ranges.items():
        cnt = int(np.sum(mask))
        if cnt > 0:
            out[key] = {
                "count": cnt,
                "target_mean": round(float(np.mean(targets[mask])), 2),
                "base_mean": round(float(np.mean(base_preds[mask])), 2),
                "custom_mean": round(float(np.mean(custom_preds[mask])), 2),
            }
        else:
            out[key] = {
                "count": 0,
                "target_mean": None,
                "base_mean": None,
                "custom_mean": None,
            }
    return out


def train_head_on_indices(
    prepared_samples: List[Dict[str, Any]],
    indices: List[int],
    t_low: float,
    t_high: float,
    seed: int = 42,
    steps: int = 120,
) -> DifferentiableKeyMatcherHead:
    torch.manual_seed(seed)
    head = DifferentiableKeyMatcherHead(t_low=t_low, t_high=t_high, hidden_dim=8)
    opt = torch.optim.AdamW(head.parameters(), lr=1.5e-2, weight_decay=1e-2)
    loss_fn = nn.SmoothL1Loss(beta=0.05)

    for _ in range(steps):
        head.train()
        opt.zero_grad()
        losses = []
        for idx in indices:
            item = prepared_samples[int(idx)]
            pred = head.forward_from_point_scores(
                item["raw_max_base"],
                item["crit_slices"],
                item["crit_weights_t"],
            )
            target_t = torch.tensor(item["target_norm"], dtype=torch.float32)
            losses.append(loss_fn(pred, target_t))
        torch.stack(losses).mean().backward()
        opt.step()

    head.eval()
    return head


def save_model_version(
    conn,
    user_id: int,
    model_id: int,
    training_mode: str,
    sample_count: int,
    train_count: int,
    test_count: int,
    mae_custom: float,
    pearson_custom: float,
    mae_base: float,
    pearson_base: float,
    head_module: DifferentiableKeyMatcherHead,
    peft_encoder: nn.Module = None,
    version_metadata: Dict[str, Any] = None,
) -> Tuple[int, int]:
    with conn.cursor() as cur:
        cur.execute(
            "SELECT COALESCE(MAX(version_num), 0) + 1 AS next_v FROM custom_model_versions WHERE model_id = %s",
            (model_id,),
        )
        version_num = int(cur.fetchone()["next_v"])

    storage_path = os.path.join(
        Config.STORAGE_DIR,
        "models",
        f"user_{user_id}",
        f"model_{model_id}",
        f"v{version_num}",
    )
    os.makedirs(storage_path, exist_ok=True)
    torch.save(head_module.state_dict(), os.path.join(storage_path, "matcher_head.pt"))

    if peft_encoder is not None and training_mode == "lora":
        adapter_dir = os.path.join(storage_path, "adapter")
        os.makedirs(adapter_dir, exist_ok=True)
        peft_encoder.save_pretrained(adapter_dir)

    meta_payload = version_metadata or {}
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO custom_model_versions (
                model_id, version_num, training_mode, storage_path,
                sample_count, train_count, test_count,
                mae_custom, pearson_custom, mae_base, pearson_base,
                metadata, is_active
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, FALSE)
            RETURNING id
            """,
            (
                model_id,
                version_num,
                training_mode,
                storage_path,
                sample_count,
                train_count,
                test_count,
                mae_custom,
                pearson_custom,
                mae_base,
                pearson_base,
                Json(meta_payload),
            ),
        )
        new_id = int(cur.fetchone()["id"])
    conn.commit()
    return new_id, version_num


def process_training_job(r, payload: Dict[str, Any]):
    num_threads = max(1, int(Config.TRAINER_THREADS))
    torch.set_num_threads(num_threads)

    job_id = str(payload["job_id"])
    model_id = int(payload["model_id"])
    use_lora = bool(payload.get("use_lora", False))
    start_ts = time.time()

    print(
        f"[TRAINER] Memulai training job={job_id} untuk model_id={model_id} "
        f"(use_lora={use_lora}, CPU threads={torch.get_num_threads()})",
        flush=True,
    )

    conn = get_db_connection()
    try:
        update_training_job_progress(
            conn,
            r,
            job_id,
            "processing",
            "menyiapkan data latih",
            5,
            metadata={"use_lora": use_lora},
        )

        with conn.cursor() as cur:
            cur.execute("SELECT t_high, t_low, min_train_samples FROM ai_settings WHERE id = 1")
            ai_cfg = cur.fetchone()
            t_high = float(ai_cfg["t_high"]) if ai_cfg else 0.60
            t_low = float(ai_cfg["t_low"]) if ai_cfg else 0.45
            min_samples = int(ai_cfg["min_train_samples"]) if ai_cfg else Config.MIN_TRAIN_SAMPLES

            cur.execute("SELECT id, user_id, name FROM custom_models WHERE id = %s", (model_id,))
            custom_model = cur.fetchone()
            if not custom_model:
                raise RuntimeError(f"Model custom ID {model_id} tidak ditemukan.")

            cur.execute(
                """
                SELECT a.id, a.title, a.max_score
                FROM custom_model_assignments cma
                JOIN assignments a ON a.id = cma.assignment_id
                WHERE cma.model_id = %s
                ORDER BY a.id ASC
                """,
                (model_id,),
            )
            linked_assignments = cur.fetchall()
            if not linked_assignments:
                raise RuntimeError("Belum ada tugas sumber data latih yang dipilih untuk model custom ini.")

            assignment_map = {}
            for a_row in linked_assignments:
                aid = a_row["id"]
                cur.execute(
                    """
                    SELECT id, name, weight, answer_key
                    FROM rubric_criteria
                    WHERE assignment_id = %s
                    ORDER BY sort_order ASC, id ASC
                    """,
                    (aid,),
                )
                c_rows = cur.fetchall()
                assignment_map[aid] = {
                    "id": aid,
                    "title": a_row["title"],
                    "max_score": float(a_row["max_score"]) if float(a_row["max_score"]) > 0 else 100.0,
                    "criteria_rows": c_rows,
                }

            assignment_ids = tuple(assignment_map.keys())
            cur.execute(
                """
                SELECT id, assignment_id, student_name, answer_text, target_score
                FROM training_samples
                WHERE assignment_id IN %s
                ORDER BY id ASC
                """,
                (assignment_ids,),
            )
            raw_samples = cur.fetchall()

        if len(raw_samples) < min_samples:
            raise RuntimeError(
                f"Jumlah data latih belum mencukupi ({len(raw_samples)}/{min_samples} sampel minimum)."
            )

        engine = EmbeddingEngine.get_instance()
        tokenizer = engine.tokenizer

        for aid, a_info in assignment_map.items():
            criteria_data = []
            all_kps = []
            crit_slices = []
            crit_weights = []
            cur_idx = 0
            for c_row in a_info["criteria_rows"]:
                pts = split_key_points(c_row["answer_key"])
                vecs = engine.encode(pts) if pts else np.zeros((0, 768), dtype=np.float32)
                w = float(c_row["weight"])
                criteria_data.append(
                    {
                        "id": c_row["id"],
                        "name": c_row["name"],
                        "weight": w,
                        "key_points": pts,
                        "key_vecs": vecs,
                    }
                )
                all_kps.extend(pts)
                crit_slices.append((cur_idx, cur_idx + len(pts)))
                cur_idx += len(pts)
                crit_weights.append(w)

            kp_enc = tokenizer(
                all_kps if all_kps else [""],
                padding=True,
                truncation=True,
                max_length=512,
                return_tensors="pt",
            )
            a_info["criteria_data"] = criteria_data
            a_info["all_kps"] = all_kps
            a_info["crit_slices"] = crit_slices
            a_info["crit_weights_t"] = torch.tensor(crit_weights, dtype=torch.float32)
            a_info["kp_input_ids"] = kp_enc["input_ids"]
            a_info["kp_attention_mask"] = kp_enc["attention_mask"]

        update_training_job_progress(
            conn,
            r,
            job_id,
            "processing",
            "ekstraksi fitur 8 dimensi",
            12,
            metadata={"use_lora": use_lora},
        )

        prepared_samples = []
        for s_row in raw_samples:
            aid = s_row["assignment_id"]
            a_info = assignment_map[aid]
            max_score = a_info["max_score"]
            ans_text = s_row["answer_text"]

            res, _ = score_single_submission(
                engine=engine,
                answer_text=ans_text,
                criteria_data=a_info["criteria_data"],
                t_high=t_high,
                t_low=t_low,
                max_score=max_score,
            )
            base_norm = float(np.clip(res["aggregate_features"][0], 0.0, 1.0))
            target_norm = float(np.clip(float(s_row["target_score"]) / max_score, 0.0, 1.0))

            windows = build_answer_windows(ans_text, tokenizer=tokenizer, max_tokens=512)
            if not windows:
                windows = [""]
            win_enc = tokenizer(
                windows, padding=True, truncation=True, max_length=512, return_tensors="pt"
            )

            prepared_samples.append(
                {
                    "id": s_row["id"],
                    "assignment_id": aid,
                    "student_name": s_row["student_name"],
                    "win_input_ids": win_enc["input_ids"],
                    "win_attention_mask": win_enc["attention_mask"],
                    "kp_input_ids": a_info["kp_input_ids"],
                    "kp_attention_mask": a_info["kp_attention_mask"],
                    "raw_max_base": torch.tensor(res["raw_point_scores"], dtype=torch.float32),
                    "crit_slices": a_info["crit_slices"],
                    "crit_weights_t": a_info["crit_weights_t"],
                    "base_norm": base_norm,
                    "target_norm": target_norm,
                }
            )

        n_samples = len(prepared_samples)
        n_splits = 5
        y_all = np.array([item["target_norm"] for item in prepared_samples])
        strata = np.digitize(y_all, np.linspace(0, 1.0001, 4))
        try:
            skf = StratifiedKFold(n_splits=n_splits, shuffle=True, random_state=42)
            splits = list(skf.split(prepared_samples, strata))
        except ValueError:
            kf = KFold(n_splits=n_splits, shuffle=True, random_state=42)
            splits = list(kf.split(prepared_samples))

        avg_train_count = int(round(np.mean([len(tr) for tr, _ in splits])))
        total_cv_test_count = n_samples

        # =========================================================================
        # TAHAP 1: Evaluasi 5-Fold CV Mode (b) "Kalibrasi" (Selalu Dijalankan)
        # =========================================================================
        t0_cal = time.time()
        fold_mae_base, fold_r_base = [], []
        fold_mae_cal, fold_r_cal = [], []
        oof_targets = np.array([item["target_norm"] * 100.0 for item in prepared_samples])
        oof_base = np.array([item["base_norm"] * 100.0 for item in prepared_samples])
        oof_cal = np.zeros(n_samples, dtype=np.float64)
        cal_fold_heads = []

        for fold_idx, (tr_idx, te_idx) in enumerate(splits, start=1):
            pct = 15 + int((fold_idx / n_splits) * (15 if use_lora else 70))
            update_training_job_progress(
                conn,
                r,
                job_id,
                "processing",
                f"Kalibrasi 5-Fold CV: Fold {fold_idx}/{n_splits}",
                pct,
                metadata={
                    "mode": "calibration",
                    "fold": fold_idx,
                    "total_folds": n_splits,
                    "use_lora": use_lora,
                },
            )

            head_f = train_head_on_indices(
                prepared_samples,
                list(tr_idx),
                t_low=t_low,
                t_high=t_high,
                seed=42 + fold_idx - 1,
                steps=120,
            )
            cal_fold_heads.append(copy.deepcopy(head_f.state_dict()))

            yt_f, yb_f, yc_f = [], [], []
            with torch.no_grad():
                for i in te_idx:
                    item = prepared_samples[int(i)]
                    pred_norm = float(
                        head_f.forward_from_point_scores(
                            item["raw_max_base"],
                            item["crit_slices"],
                            item["crit_weights_t"],
                        ).item()
                    )
                    pred_100 = float(np.clip(pred_norm, 0.0, 1.0)) * 100.0
                    oof_cal[int(i)] = pred_100
                    yt_f.append(item["target_norm"] * 100.0)
                    yb_f.append(item["base_norm"] * 100.0)
                    yc_f.append(pred_100)

            yt_arr = np.array(yt_f)
            yb_arr = np.array(yb_f)
            yc_arr = np.array(yc_f)
            fold_mae_base.append(float(np.mean(np.abs(yt_arr - yb_arr))))
            fold_r_base.append(safe_pearson(yt_arr, yb_arr))
            fold_mae_cal.append(float(np.mean(np.abs(yt_arr - yc_arr))))
            fold_r_cal.append(safe_pearson(yt_arr, yc_arr))

        cv_mae_base = round(float(np.mean(fold_mae_base)), 2)
        cv_r_base = round(float(np.mean(fold_r_base)), 4)
        cv_mae_cal = round(float(np.mean(fold_mae_cal)), 2)
        cv_r_cal = round(float(np.mean(fold_r_cal)), 4)

        # Latih ulang model final Kalibrasi dengan seluruh data (100%)
        update_training_job_progress(
            conn,
            r,
            job_id,
            "processing",
            "Kalibrasi: Melatih model final (100% data)",
            32 if use_lora else 90,
            metadata={"mode": "calibration", "cv_mae": cv_mae_cal, "use_lora": use_lora},
        )
        final_cal_head = train_head_on_indices(
            prepared_samples,
            list(range(n_samples)),
            t_low=t_low,
            t_high=t_high,
            seed=42,
            steps=120,
        )
        elapsed_cal = round(time.time() - t0_cal, 2)

        cal_meta = {
            "evaluation_method": "5-Fold Cross-Validation",
            "n_splits": n_splits,
            "elapsed_seconds": elapsed_cal,
            "fold_mae_custom": [round(x, 2) for x in fold_mae_cal],
            "fold_mae_base": [round(x, 2) for x in fold_mae_base],
            "range_summary": compute_range_summary(oof_targets, oof_base, oof_cal),
        }

        cal_v_id, cal_v_num = save_model_version(
            conn=conn,
            user_id=custom_model["user_id"],
            model_id=model_id,
            training_mode="calibration",
            sample_count=n_samples,
            train_count=avg_train_count,
            test_count=total_cv_test_count,
            mae_custom=cv_mae_cal,
            pearson_custom=cv_r_cal,
            mae_base=cv_mae_base,
            pearson_base=cv_r_base,
            head_module=final_cal_head,
            peft_encoder=None,
            version_metadata=cal_meta,
        )

        created_versions = [
            {
                "version_id": cal_v_id,
                "version_num": cal_v_num,
                "training_mode": "calibration",
                "mae_custom": cv_mae_cal,
                "pearson_custom": cv_r_cal,
                "elapsed_seconds": elapsed_cal,
            }
        ]

        # =========================================================================
        # TAHAP 2 (Opsional): Jika Guru Mencentang "Fine-tune mendalam (LoRA)"
        # =========================================================================
        if use_lora:
            t0_lora = time.time()
            fold_mae_lora, fold_r_lora = [], []
            oof_lora = np.zeros(n_samples, dtype=np.float64)
            lora_epochs = 3
            loss_fn = nn.SmoothL1Loss(beta=0.05)

            for fold_idx, (tr_idx, te_idx) in enumerate(splits, start=1):
                torch.manual_seed(42 + fold_idx - 1)
                rng = np.random.default_rng(42 + fold_idx - 1)

                head_l = DifferentiableKeyMatcherHead(t_low=t_low, t_high=t_high, hidden_dim=8)
                head_l.load_state_dict(cal_fold_heads[fold_idx - 1])

                base_copy = copy.deepcopy(engine.model)
                lora_cfg = LoraConfig(
                    r=8,
                    lora_alpha=16,
                    lora_dropout=0.0,
                    bias="none",
                    target_modules=["query", "key", "value", "dense"],
                )
                peft_enc = get_peft_model(base_copy, lora_cfg)
                disable_bert_dropout(peft_enc)
                net_l = CustomKeyMatcherModel(peft_enc, head_l)

                opt_l = torch.optim.AdamW(
                    [
                        {"params": peft_enc.parameters(), "lr": 1e-4, "weight_decay": 1e-2},
                        {"params": head_l.parameters(), "lr": 1e-3, "weight_decay": 1e-2},
                    ]
                )

                for ep in range(1, lora_epochs + 1):
                    step_num = (fold_idx - 1) * lora_epochs + ep
                    total_steps = n_splits * lora_epochs
                    pct = 35 + int((step_num / total_steps) * 50)
                    update_training_job_progress(
                        conn,
                        r,
                        job_id,
                        "processing",
                        f"LoRA 5-Fold CV: Fold {fold_idx}/{n_splits} — Epoch {ep}/{lora_epochs}",
                        pct,
                        metadata={
                            "mode": "lora",
                            "fold": fold_idx,
                            "total_folds": n_splits,
                            "epoch": ep,
                            "total_epochs": lora_epochs,
                            "use_lora": True,
                        },
                    )

                    net_l.train()
                    perm = rng.permutation(tr_idx)
                    opt_l.zero_grad()
                    for step_i, i in enumerate(perm, start=1):
                        item = prepared_samples[int(i)]
                        pred = net_l.forward_single(
                            item["win_input_ids"],
                            item["win_attention_mask"],
                            item["kp_input_ids"],
                            item["kp_attention_mask"],
                            item["crit_slices"],
                            item["crit_weights_t"],
                        )
                        target_t = torch.tensor(item["target_norm"], dtype=torch.float32)
                        loss = loss_fn(pred, target_t)
                        loss.backward()
                        if step_i % 6 == 0 or step_i == len(perm):
                            torch.nn.utils.clip_grad_norm_(net_l.parameters(), 1.0)
                            opt_l.step()
                            opt_l.zero_grad()

                net_l.eval()
                yt_f, yl_f = [], []
                with torch.no_grad():
                    for i in te_idx:
                        item = prepared_samples[int(i)]
                        pred_norm = float(
                            net_l.forward_single(
                                item["win_input_ids"],
                                item["win_attention_mask"],
                                item["kp_input_ids"],
                                item["kp_attention_mask"],
                                item["crit_slices"],
                                item["crit_weights_t"],
                            ).item()
                        )
                        pred_100 = float(np.clip(pred_norm, 0.0, 1.0)) * 100.0
                        oof_lora[int(i)] = pred_100
                        yt_f.append(item["target_norm"] * 100.0)
                        yl_f.append(pred_100)

                yt_arr = np.array(yt_f)
                yl_arr = np.array(yl_f)
                fold_mae_lora.append(float(np.mean(np.abs(yt_arr - yl_arr))))
                fold_r_lora.append(safe_pearson(yt_arr, yl_arr))

            cv_mae_lora = round(float(np.mean(fold_mae_lora)), 2)
            cv_r_lora = round(float(np.mean(fold_r_lora)), 4)

            # Latih ulang model final LoRA dengan seluruh data (100%)
            torch.manual_seed(42)
            rng_full = np.random.default_rng(42)
            final_lora_head = copy.deepcopy(final_cal_head)
            base_full = copy.deepcopy(engine.model)
            lora_cfg_full = LoraConfig(
                r=8,
                lora_alpha=16,
                lora_dropout=0.0,
                bias="none",
                target_modules=["query", "key", "value", "dense"],
            )
            peft_full = get_peft_model(base_full, lora_cfg_full)
            disable_bert_dropout(peft_full)
            net_full = CustomKeyMatcherModel(peft_full, final_lora_head)

            opt_full = torch.optim.AdamW(
                [
                    {"params": peft_full.parameters(), "lr": 1e-4, "weight_decay": 1e-2},
                    {"params": final_lora_head.parameters(), "lr": 1e-3, "weight_decay": 1e-2},
                ]
            )

            for ep in range(1, lora_epochs + 1):
                pct = 86 + int((ep / lora_epochs) * 10)
                update_training_job_progress(
                    conn,
                    r,
                    job_id,
                    "processing",
                    f"LoRA: Melatih model final (Epoch {ep}/{lora_epochs})",
                    pct,
                    metadata={
                        "mode": "lora_final",
                        "epoch": ep,
                        "total_epochs": lora_epochs,
                        "cv_mae_calibration": cv_mae_cal,
                        "cv_mae_lora": cv_mae_lora,
                    },
                )
                net_full.train()
                perm = rng_full.permutation(n_samples)
                opt_full.zero_grad()
                for step_i, i in enumerate(perm, start=1):
                    item = prepared_samples[int(i)]
                    pred = net_full.forward_single(
                        item["win_input_ids"],
                        item["win_attention_mask"],
                        item["kp_input_ids"],
                        item["kp_attention_mask"],
                        item["crit_slices"],
                        item["crit_weights_t"],
                    )
                    target_t = torch.tensor(item["target_norm"], dtype=torch.float32)
                    loss = loss_fn(pred, target_t)
                    loss.backward()
                    if step_i % 6 == 0 or step_i == len(perm):
                        torch.nn.utils.clip_grad_norm_(net_full.parameters(), 1.0)
                        opt_full.step()
                        opt_full.zero_grad()

            net_full.eval()
            elapsed_lora = round(time.time() - t0_lora, 2)

            lora_meta = {
                "evaluation_method": "5-Fold Cross-Validation",
                "n_splits": n_splits,
                "elapsed_seconds": elapsed_lora,
                "fold_mae_custom": [round(x, 2) for x in fold_mae_lora],
                "fold_mae_base": [round(x, 2) for x in fold_mae_base],
                "range_summary": compute_range_summary(oof_targets, oof_base, oof_lora),
                "compared_with_calibration_version": cal_v_num,
                "calibration_mae_cv": cv_mae_cal,
            }

            lora_v_id, lora_v_num = save_model_version(
                conn=conn,
                user_id=custom_model["user_id"],
                model_id=model_id,
                training_mode="lora",
                sample_count=n_samples,
                train_count=avg_train_count,
                test_count=total_cv_test_count,
                mae_custom=cv_mae_lora,
                pearson_custom=cv_r_lora,
                mae_base=cv_mae_base,
                pearson_base=cv_r_base,
                head_module=final_lora_head,
                peft_encoder=peft_full,
                version_metadata=lora_meta,
            )

            created_versions.append(
                {
                    "version_id": lora_v_id,
                    "version_num": lora_v_num,
                    "training_mode": "lora",
                    "mae_custom": cv_mae_lora,
                    "pearson_custom": cv_r_lora,
                    "elapsed_seconds": elapsed_lora,
                }
            )

        best_v = min(created_versions, key=lambda x: x["mae_custom"])
        total_elapsed = round(time.time() - start_ts, 2)

        update_training_job_progress(
            conn,
            r,
            job_id,
            "completed",
            "selesai",
            100,
            metadata={
                "use_lora": use_lora,
                "created_versions": created_versions,
                "recommended_version_id": best_v["version_id"],
                "recommended_version_num": best_v["version_num"],
                "recommended_mode": best_v["training_mode"],
                "mae_base": cv_mae_base,
                "pearson_base": cv_r_base,
                "elapsed_seconds": total_elapsed,
            },
        )
        print(
            f"[TRAINER] Selesai job={job_id} dalam {total_elapsed}s: "
            f"versi dibuat={created_versions}, rekomendasi=v{best_v['version_num']} ({best_v['training_mode']})",
            flush=True,
        )
    except Exception as exc:
        traceback.print_exc()
        try:
            conn.rollback()
            update_training_job_progress(conn, r, job_id, "failed", "gagal", 100, error_message=str(exc))
        except Exception:
            pass
    finally:
        conn.close()
