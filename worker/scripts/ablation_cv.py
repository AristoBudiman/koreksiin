"""
Skrip Ablation 5-Fold Cross-Validation Koreksi.in
Membandingkan:
  (a) Model Bawaan (LazarusNLP/all-indobert-base-v4)
  (b) Mode Kalibrasi (DifferentiableKeyMatcherHead di atas 8 fitur agregat baru, tanpa LoRA)
  (c) Mode Fine-tune mendalam (LoRA + DifferentiableKeyMatcherHead)

Cara menjalankan di dalam container worker:
    docker compose exec worker python -m scripts.ablation_cv
"""
import copy
import os
import time
import numpy as np
import openpyxl
import torch
import torch.nn as nn
from peft import LoraConfig, get_peft_model
from sklearn.model_selection import StratifiedKFold

from app.preprocess import split_key_points, build_answer_windows
from app.scorer import EmbeddingEngine, score_single_submission
from app.trainer import (
    DifferentiableKeyMatcherHead,
    CustomKeyMatcherModel,
    disable_bert_dropout,
    safe_pearson,
)

SAMPLE_PATH = os.getenv("SAMPLE_EXCEL_PATH", "/out_samples/contoh_fotosintesis_30_siswa.xlsx")


def main():
    torch.set_num_threads(2)
    torch.manual_seed(42)
    np.random.seed(42)

    if not os.path.exists(SAMPLE_PATH):
        import app.generate_samples  # noqa: F401

    wb = openpyxl.load_workbook(SAMPLE_PATH)
    ws = wb["Jawaban Siswa"]
    students = []
    for idx, row in enumerate(ws.iter_rows(min_row=2, values_only=True), start=1):
        if row[0] and row[1]:
            students.append(
                {
                    "idx": idx,
                    "name": str(row[0]),
                    "answer": str(row[1]),
                    "target": float(row[2]),
                }
            )

    criteria = [
        {
            "id": 1,
            "name": "Definisi Proses Fotosintesis",
            "weight": 35.0,
            "key": (
                "Fotosintesis adalah proses pembuatan atau pembentukan makanan berupa glukosa oleh tumbuhan hijau.\n"
                "Tumbuhan hijau menggunakan bantuan energi cahaya matahari."
            ),
        },
        {
            "id": 2,
            "name": "Bahan Baku dan Hasil",
            "weight": 35.0,
            "key": (
                "Tumbuhan membutuhkan bahan baku berupa karbon dioksida (CO2) dan air (H2O).\n"
                "Reaksi ini menghasilkan glukosa dan gas oksigen (O2)."
            ),
        },
        {
            "id": 3,
            "name": "Tempat Terjadinya dan Peran Klorofil",
            "weight": 30.0,
            "key": (
                "Reaksi berlangsung di dalam organel kloroplas pada daun.\n"
                "Daun memiliki pigmen klorofil untuk menangkap atau menyerap cahaya matahari."
            ),
        },
    ]

    engine = EmbeddingEngine.get_instance()
    tokenizer = engine.tokenizer

    all_kps = []
    crit_slices = []
    crit_weights = []
    criteria_data = []
    cur_idx = 0
    for c in criteria:
        pts = split_key_points(c["key"])
        vecs = engine.encode(pts)
        criteria_data.append(
            {
                "id": c["id"],
                "name": c["name"],
                "weight": c["weight"],
                "key_points": pts,
                "key_vecs": vecs,
            }
        )
        all_kps.extend(pts)
        crit_slices.append((cur_idx, cur_idx + len(pts)))
        cur_idx += len(pts)
        crit_weights.append(c["weight"])

    crit_weights_t = torch.tensor(crit_weights, dtype=torch.float32)
    kp_encoded = tokenizer(
        all_kps, padding=True, truncation=True, max_length=512, return_tensors="pt"
    )
    kp_vecs_base = engine.encode(all_kps)

    prepared = []
    for st in students:
        res, _ = score_single_submission(engine, st["answer"], criteria_data, 0.60, 0.45, 100.0)
        wins = build_answer_windows(st["answer"], tokenizer=tokenizer, max_tokens=512)
        if not wins:
            wins = [""]
        win_enc = tokenizer(
            wins, padding=True, truncation=True, max_length=512, return_tensors="pt"
        )
        win_vecs_base = engine.encode(wins)
        sim_mat = np.matmul(kp_vecs_base, win_vecs_base.T)
        raw_max_base = torch.tensor(np.max(sim_mat, axis=1), dtype=torch.float32)

        prepared.append(
            {
                "idx": st["idx"],
                "name": st["name"],
                "win_input_ids": win_enc["input_ids"],
                "win_attention_mask": win_enc["attention_mask"],
                "raw_max_base": raw_max_base,
                "base_pred": float(res["predicted_score"]),
                "target_norm": float(st["target"]) / 100.0,
                "target": float(st["target"]),
            }
        )

    y_all = np.array([p["target_norm"] for p in prepared])
    strata = np.digitize(y_all, np.linspace(0, 1.0001, 4))
    skf = StratifiedKFold(n_splits=5, shuffle=True, random_state=42)
    splits = list(skf.split(prepared, strata))

    # (a) Model Bawaan
    fold_mae_a, fold_r_a = [], []
    oof_preds_a = np.zeros(len(prepared))
    for tr_idx, te_idx in splits:
        yt = np.array([prepared[i]["target"] for i in te_idx])
        yp = np.array([prepared[i]["base_pred"] for i in te_idx])
        for i in te_idx:
            oof_preds_a[i] = prepared[i]["base_pred"]
        fold_mae_a.append(np.mean(np.abs(yt - yp)))
        fold_r_a.append(safe_pearson(yt, yp))

    # (b) Mode Kalibrasi (Head Saja di atas 8 fitur agregat baru, tanpa LoRA)
    t0_b = time.time()
    fold_mae_b, fold_r_b = [], []
    oof_preds_b = np.zeros(len(prepared))
    trained_heads_b = []
    loss_fn = nn.SmoothL1Loss(beta=0.05)

    for fold_i, (tr_idx, te_idx) in enumerate(splits):
        torch.manual_seed(42 + fold_i)
        head_b = DifferentiableKeyMatcherHead(t_low=0.45, t_high=0.60, hidden_dim=8)
        opt_b = torch.optim.AdamW(head_b.parameters(), lr=1.5e-2, weight_decay=1e-2)

        for _ in range(120):
            head_b.train()
            opt_b.zero_grad()
            losses = [
                loss_fn(
                    head_b.forward_from_point_scores(
                        prepared[i]["raw_max_base"], crit_slices, crit_weights_t
                    ),
                    torch.tensor(prepared[i]["target_norm"], dtype=torch.float32),
                )
                for i in tr_idx
            ]
            torch.stack(losses).mean().backward()
            opt_b.step()

        trained_heads_b.append(copy.deepcopy(head_b.state_dict()))
        head_b.eval()
        yt, yp = [], []
        with torch.no_grad():
            for i in te_idx:
                p = (
                    float(
                        head_b.forward_from_point_scores(
                            prepared[i]["raw_max_base"], crit_slices, crit_weights_t
                        ).item()
                    )
                    * 100.0
                )
                oof_preds_b[i] = p
                yt.append(prepared[i]["target"])
                yp.append(p)
        fold_mae_b.append(np.mean(np.abs(np.array(yt) - np.array(yp))))
        fold_r_b.append(safe_pearson(np.array(yt), np.array(yp)))
    elapsed_b = time.time() - t0_b

    # (c) Mode Fine-tune mendalam (LoRA + Head)
    t0_c = time.time()
    fold_mae_c, fold_r_c = [], []
    oof_preds_c = np.zeros(len(prepared))

    for fold_i, (tr_idx, te_idx) in enumerate(splits):
        torch.manual_seed(42 + fold_i)
        rng = np.random.default_rng(42 + fold_i)

        head_c = DifferentiableKeyMatcherHead(t_low=0.45, t_high=0.60, hidden_dim=8)
        head_c.load_state_dict(trained_heads_b[fold_i])

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
        model_c = CustomKeyMatcherModel(peft_enc, head_c)

        opt_c = torch.optim.AdamW(
            [
                {"params": peft_enc.parameters(), "lr": 1e-4, "weight_decay": 1e-2},
                {"params": head_c.parameters(), "lr": 1e-3, "weight_decay": 1e-2},
            ]
        )

        for _ in range(3):
            model_c.train()
            perm = rng.permutation(tr_idx)
            opt_c.zero_grad()
            for step_i, i in enumerate(perm, 1):
                item = prepared[int(i)]
                pred = model_c.forward_single(
                    item["win_input_ids"],
                    item["win_attention_mask"],
                    kp_encoded["input_ids"],
                    kp_encoded["attention_mask"],
                    crit_slices,
                    crit_weights_t,
                )
                target_t = torch.tensor(item["target_norm"], dtype=torch.float32)
                loss = loss_fn(pred, target_t)
                loss.backward()
                if step_i % 6 == 0 or step_i == len(perm):
                    torch.nn.utils.clip_grad_norm_(model_c.parameters(), 1.0)
                    opt_c.step()
                    opt_c.zero_grad()

        model_c.eval()
        yt, yp = [], []
        with torch.no_grad():
            for i in te_idx:
                item = prepared[i]
                p = (
                    float(
                        model_c.forward_single(
                            item["win_input_ids"],
                            item["win_attention_mask"],
                            kp_encoded["input_ids"],
                            kp_encoded["attention_mask"],
                            crit_slices,
                            crit_weights_t,
                        ).item()
                    )
                    * 100.0
                )
                oof_preds_c[i] = p
                yt.append(item["target"])
                yp.append(p)
        mae_f = float(np.mean(np.abs(np.array(yt) - np.array(yp))))
        r_f = safe_pearson(np.array(yt), np.array(yp))
        fold_mae_c.append(mae_f)
        fold_r_c.append(r_f)
        print(
            f"  [Fold {fold_i+1}/5] Base MAE={fold_mae_a[fold_i]:.2f} (r={fold_r_a[fold_i]:.4f}) | "
            f"Kalibrasi MAE={fold_mae_b[fold_i]:.2f} (r={fold_r_b[fold_i]:.4f}) | "
            f"LoRA+Head MAE={mae_f:.2f} (r={r_f:.4f})",
            flush=True,
        )

    elapsed_c = time.time() - t0_c

    def group_stats(preds):
        g1 = preds[:10]
        g2 = preds[10:26]
        g3 = preds[26:30]
        return (
            f"G1(1-10)={np.mean(g1):.2f} [{np.min(g1):.1f}-{np.max(g1):.1f}] | "
            f"G2(11-26)={np.mean(g2):.2f} [{np.min(g2):.1f}-{np.max(g2):.1f}] | "
            f"G3(27-30)={np.mean(g3):.2f} [{np.min(g3):.1f}-{np.max(g3):.1f}]"
        )

    print("\n================ HASIL ABLATION 5-FOLD CROSS VALIDATION ================")
    print(
        f"(a) Model Bawaan        : CV MAE = {np.mean(fold_mae_a):.2f} | CV Pearson = {np.mean(fold_r_a):.4f} | {group_stats(oof_preds_a)}"
    )
    print(
        f"(b) Kalibrasi (No LoRA) : CV MAE = {np.mean(fold_mae_b):.2f} | CV Pearson = {np.mean(fold_r_b):.4f} | Time={elapsed_b:.2f}s | {group_stats(oof_preds_b)}"
    )
    print(
        f"(c) LoRA + Head         : CV MAE = {np.mean(fold_mae_c):.2f} | CV Pearson = {np.mean(fold_r_c):.4f} | Time={elapsed_c:.2f}s | {group_stats(oof_preds_c)}"
    )


if __name__ == "__main__":
    main()

