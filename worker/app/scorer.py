from typing import List, Dict, Any, Tuple
import numpy as np
import torch
import torch.nn.functional as F
from transformers import AutoTokenizer, AutoModel
from app.config import Config
from app.preprocess import split_key_points, build_answer_windows


class EmbeddingEngine:
    _instance = None

    def __init__(self, model_name: str = None):
        target_model = model_name or Config.BASE_MODEL_NAME
        print(f"[EmbeddingEngine] Memuat model dasar: {target_model}", flush=True)
        self.tokenizer = AutoTokenizer.from_pretrained(target_model)
        self.model = AutoModel.from_pretrained(target_model)
        self.model.to("cpu")
        self.model.eval()
        print("[EmbeddingEngine] Model dasar berhasil dimuat di CPU.", flush=True)

    @classmethod
    def get_instance(cls) -> "EmbeddingEngine":
        if cls._instance is None:
            cls._instance = cls()
        return cls._instance

    @torch.no_grad()
    def encode(self, texts: List[str], batch_size: int = 16) -> np.ndarray:
        """
        Menghitung sentence embedding dengan mean pooling sesuai model card
        LazarusNLP/all-indobert-base-v4 diikuti normalisasi L2.
        """
        if not texts:
            return np.zeros((0, 768), dtype=np.float32)

        all_vecs = []
        for start in range(0, len(texts), batch_size):
            batch = texts[start : start + batch_size]
            encoded = self.tokenizer(
                batch,
                padding=True,
                truncation=True,
                max_length=512,
                return_tensors="pt",
            )
            outputs = self.model(**encoded)
            token_embeddings = outputs[0]  # last_hidden_state [B, L, H]
            input_mask_expanded = (
                encoded["attention_mask"]
                .unsqueeze(-1)
                .expand(token_embeddings.size())
                .float()
            )
            sum_embeddings = torch.sum(token_embeddings * input_mask_expanded, 1)
            sum_mask = torch.clamp(input_mask_expanded.sum(1), min=1e-9)
            mean_pooled = sum_embeddings / sum_mask
            normalized = F.normalize(mean_pooled, p=2, dim=1)
            all_vecs.append(normalized.cpu().numpy())

        return np.vstack(all_vecs).astype(np.float32)


def rescale_point_score(raw_s: float, t_low: float, t_high: float) -> float:
    """
    Skalakan ulang skor mentah poin kunci:
    s' = clip((s - T_LOW) / (T_HIGH - T_LOW), 0, 1)
    """
    denom = max(t_high - t_low, 1e-6)
    return float(np.clip((raw_s - t_low) / denom, 0.0, 1.0))


def classify_point_status(raw_s: float, t_low: float, t_high: float) -> str:
    """
    Status poin kunci:
    - 'tercakup' kalau s >= T_HIGH
    - 'sebagian' kalau T_LOW <= s < T_HIGH
    - 'belum_ada' kalau s < T_LOW
    """
    if raw_s >= t_high:
        return "tercakup"
    elif raw_s >= t_low:
        return "sebagian"
    return "belum_ada"


def clean_point_label(point: str) -> str:
    s = point.strip().rstrip(".!;")
    if len(s) > 1 and s[0].isupper() and not s[:3].isupper():
        s = s[0].lower() + s[1:]
    return s


def build_criterion_feedback(
    criterion_name: str,
    status: str,
    all_points: List[str],
    missing_points: List[str],
) -> str:
    """
    Menyusun kalimat feedback template berdasarkan status kriteria dan
    poin kunci yang belum terpenuhi penuh (tanpa LLM dan tanpa emoji).
    """
    name = criterion_name.strip()
    if status == "terpenuhi":
        return f"{name} sudah tepat dan lengkap."
    elif status == "sebagian":
        if missing_points:
            joined = "; ".join(clean_point_label(p) for p in missing_points)
            return f"{name} baru dijelaskan sebagian. Belum menyebut: {joined}."
        return f"{name} baru dijelaskan sebagian."
    else:
        target_pts = all_points if all_points else missing_points
        if target_pts:
            joined = "; ".join(clean_point_label(p) for p in target_pts)
            return f"{name} belum dijelaskan. Seharusnya menyebut: {joined}."
        return f"{name} belum dijelaskan."


def compute_fixed_aggregate_features(
    criteria_results: List[Dict[str, Any]],
    weights: List[float],
) -> List[float]:
    """
    Menghasilkan 6 fitur agregat berukuran tetap dari C_i hasil skala ulang:
    1. weighted_avg_coverage
    2. min_coverage
    3. max_coverage
    4. prop_terpenuhi
    5. prop_tidak_ada
    6. base_formula_score (ternormalisasi 0-1)
    """
    if not criteria_results:
        return [0.0, 0.0, 0.0, 0.0, 1.0, 0.0]

    covs = [float(c["coverage_score"]) for c in criteria_results]
    total_w = sum(weights) if sum(weights) > 0 else float(len(covs))
    weighted_avg = sum(w * c for w, c in zip(weights, covs)) / total_w
    min_cov = min(covs)
    max_cov = max(covs)
    k = len(criteria_results)
    prop_terpenuhi = sum(1 for c in criteria_results if c["status"] == "terpenuhi") / k
    prop_tidak_ada = sum(1 for c in criteria_results if c["status"] == "tidak_ada") / k
    base_formula_score = float(np.clip(weighted_avg, 0.0, 1.0))

    return [
        float(weighted_avg),
        float(min_cov),
        float(max_cov),
        float(prop_terpenuhi),
        float(prop_tidak_ada),
        float(base_formula_score),
    ]


def score_single_submission(
    engine: EmbeddingEngine,
    answer_text: str,
    criteria_data: List[Dict[str, Any]],
    t_high: float,
    t_low: float,
    max_score: float,
) -> Tuple[Dict[str, Any], np.ndarray]:
    """
    Menilai satu jawaban siswa terhadap seluruh kriteria rubrik.
    - Setiap poin kunci mendapat skor mentah maksimum s dari potongan jawaban.
    - Status poin kunci: tercakup (s >= T_HIGH), sebagian (T_LOW <= s < T_HIGH), belum_ada (s < T_LOW).
    - Skor poin kunci diskalakan ulang: s' = clip((s - T_LOW) / (T_HIGH - T_LOW), 0, 1).
    - Cakupan kriteria C_i = rata-rata s' dari poin-poin kuncinya.
    - Nilai model bawaan memakai C_i hasil skala ulang ini.
    """
    windows = build_answer_windows(answer_text, tokenizer=engine.tokenizer, max_tokens=512)
    if not windows:
        win_vecs = np.zeros((1, 768), dtype=np.float32)
        windows = [""]
    else:
        win_vecs = engine.encode(windows)

    doc_vec = np.mean(win_vecs, axis=0)
    norm = np.linalg.norm(doc_vec)
    if norm > 1e-9:
        doc_vec = doc_vec / norm

    criteria_results = []
    weights = []

    for crit in criteria_data:
        key_points = crit["key_points"]
        key_vecs = crit["key_vecs"]  # [P, 768]
        w = float(crit["weight"])
        weights.append(w)

        if len(key_points) == 0 or key_vecs.shape[0] == 0:
            criteria_results.append(
                {
                    "criterion_id": crit["id"],
                    "coverage_score": 0.0,
                    "status": "tidak_ada",
                    "matched_snippet": "",
                    "missing_key_points": [],
                    "feedback_sentence": f"{crit['name']} belum dijelaskan.",
                }
            )
            continue

        sim_matrix = np.matmul(key_vecs, win_vecs.T)
        max_indices = np.argmax(sim_matrix, axis=1)
        raw_max_scores = np.max(sim_matrix, axis=1)

        rescaled_scores = [
            rescale_point_score(float(s), t_low=t_low, t_high=t_high)
            for s in raw_max_scores
        ]
        point_statuses = [
            classify_point_status(float(s), t_low=t_low, t_high=t_high)
            for s in raw_max_scores
        ]

        coverage_score = float(np.mean(rescaled_scores))

        # Status kriteria berdasarkan status poin-poin kunci dan coverage_score (C_i di skala 0..1)
        if all(st == "tercakup" for st in point_statuses) or coverage_score >= 0.80:
            status = "terpenuhi"
        elif any(st in ("tercakup", "sebagian") for st in point_statuses) and coverage_score > 0.0:
            status = "sebagian"
        else:
            status = "tidak_ada"

        best_point_idx = int(np.argmax(raw_max_scores))
        best_win_idx = int(max_indices[best_point_idx])
        matched_snippet = windows[best_win_idx] if windows[best_win_idx] else ""

        # Poin kunci yang belum tercakup penuh (s < T_HIGH) dicatat untuk feedback
        missing_points = [
            pt for pt, st in zip(key_points, point_statuses) if st != "tercakup"
        ]
        if status == "sebagian" and not missing_points and len(key_points) > 0:
            lowest_idx = int(np.argmin(raw_max_scores))
            missing_points.append(key_points[lowest_idx])

        feedback_sentence = build_criterion_feedback(
            crit["name"], status, key_points, missing_points
        )

        criteria_results.append(
            {
                "criterion_id": crit["id"],
                "coverage_score": round(coverage_score, 4),
                "status": status,
                "matched_snippet": matched_snippet,
                "missing_key_points": missing_points,
                "feedback_sentence": feedback_sentence,
            }
        )

    agg_features = compute_fixed_aggregate_features(criteria_results, weights)
    base_norm_score = agg_features[5]
    predicted_score = round(base_norm_score * float(max_score), 2)
    feedback_text = " ".join(c["feedback_sentence"] for c in criteria_results)

    return (
        {
            "predicted_score": predicted_score,
            "feedback_text": feedback_text,
            "criteria_scores": criteria_results,
            "aggregate_features": agg_features,
        },
        doc_vec,
    )
