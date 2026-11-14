"""
Skrip Kalibrasi Ambang Empiris Koreksi.in
Cara menjalankan di dalam container worker:
    docker compose exec worker python -m scripts.calibrate_sample
"""
import json
import os
import numpy as np
import openpyxl
from app.scorer import EmbeddingEngine, score_single_submission
from app.preprocess import split_key_points, build_answer_windows
from app.similarity import compute_lexical_similarity_matrix

SAMPLE_PATH = os.getenv("SAMPLE_EXCEL_PATH", "/out_samples/contoh_fotosintesis_30_siswa.xlsx")


def main():
    if not os.path.exists(SAMPLE_PATH):
        import app.generate_samples  # noqa: F401

    wb = openpyxl.load_workbook(SAMPLE_PATH)
    ws = wb["Jawaban Siswa"]

    students = []
    for row_idx, row in enumerate(ws.iter_rows(min_row=2, values_only=True), start=1):
        name, answer, score = row[0], row[1], row[2]
        if name and answer:
            students.append({
                "idx": row_idx,
                "name": str(name),
                "answer": str(answer),
                "score": float(score) if score is not None else 0.0,
            })

    criteria = [
        {
            "id": 1,
            "name": "Definisi Proses Fotosintesis",
            "weight": 35,
            "key": (
                "Fotosintesis adalah proses pembuatan atau pembentukan makanan berupa glukosa oleh tumbuhan hijau.\n"
                "Tumbuhan hijau menggunakan bantuan energi cahaya matahari."
            ),
        },
        {
            "id": 2,
            "name": "Bahan Baku dan Hasil",
            "weight": 35,
            "key": (
                "Tumbuhan membutuhkan bahan baku berupa karbon dioksida (CO2) dan air (H2O).\n"
                "Reaksi ini menghasilkan glukosa dan gas oksigen (O2)."
            ),
        },
        {
            "id": 3,
            "name": "Tempat Terjadinya dan Peran Klorofil",
            "weight": 30,
            "key": (
                "Reaksi berlangsung di dalam organel kloroplas pada daun.\n"
                "Daun memiliki pigmen klorofil untuk menangkap atau menyerap cahaya matahari."
            ),
        },
    ]

    engine = EmbeddingEngine.get_instance()

    all_key_points = []
    for c in criteria:
        pts = split_key_points(c["key"])
        vecs = engine.encode(pts)
        c["key_points"] = pts
        c["key_vecs"] = vecs
        for p_idx, p in enumerate(pts):
            all_key_points.append({
                "crit_id": c["id"],
                "crit_name": c["name"],
                "pt": p,
                "vec": vecs[p_idx],
            })

    matched_scores = []
    unmatched_scores = []
    doc_vecs = []

    for st in students:
        wins = build_answer_windows(st["answer"], tokenizer=engine.tokenizer, max_tokens=512)
        win_vecs = engine.encode(wins)
        d_vec = np.mean(win_vecs, axis=0)
        d_vec = d_vec / (np.linalg.norm(d_vec) + 1e-9)
        doc_vecs.append(d_vec)

        for kp in all_key_points:
            sims = np.matmul(win_vecs, kp["vec"])
            max_sim = float(np.max(sims))
            if st["idx"] <= 10:
                matched_scores.append(max_sim)
            elif st["idx"] >= 27:
                unmatched_scores.append(max_sim)

    doc_matrix = np.vstack(doc_vecs)
    sem_matrix = np.matmul(doc_matrix, doc_matrix.T)
    lex_matrix = compute_lexical_similarity_matrix([s["answer"] for s in students])

    pair_sems, pair_lexs = [], []
    copied_pair_sem, copied_pair_lex = 0.0, 0.0

    n = len(students)
    for i in range(n):
        for j in range(i + 1, n):
            s_val = float(sem_matrix[i, j])
            l_val = float(lex_matrix[i, j])
            if students[i]["idx"] == 5 and students[j]["idx"] == 6:
                copied_pair_sem = s_val
                copied_pair_lex = l_val
            else:
                pair_sems.append(s_val)
                pair_lexs.append(l_val)

    t_low_val = 0.45
    t_high_val = 0.60
    g1, g2, g3 = [], [], []
    for st in students:
        res, _ = score_single_submission(
            engine=engine,
            answer_text=st["answer"],
            criteria_data=criteria,
            t_high=t_high_val,
            t_low=t_low_val,
            max_score=100.0,
        )
        pred = res["predicted_score"]
        if st["idx"] <= 10:
            g1.append(pred)
        elif st["idx"] <= 26:
            g2.append(pred)
        else:
            g3.append(pred)

    summary = {
        "thresholds": {"T_LOW": t_low_val, "T_HIGH": t_high_val, "S_SEM": 0.92, "S_LEX": 0.60},
        "matched_point_window_cosine": {
            "count": len(matched_scores),
            "min": round(float(np.min(matched_scores)), 4),
            "p25": round(float(np.percentile(matched_scores, 25)), 4),
            "median": round(float(np.median(matched_scores)), 4),
            "mean": round(float(np.mean(matched_scores)), 4),
            "p75": round(float(np.percentile(matched_scores, 75)), 4),
            "max": round(float(np.max(matched_scores)), 4),
        },
        "unmatched_point_window_cosine": {
            "count": len(unmatched_scores),
            "min": round(float(np.min(unmatched_scores)), 4),
            "p25": round(float(np.percentile(unmatched_scores, 25)), 4),
            "median": round(float(np.median(unmatched_scores)), 4),
            "mean": round(float(np.mean(unmatched_scores)), 4),
            "p75": round(float(np.percentile(unmatched_scores, 75)), 4),
            "max": round(float(np.max(unmatched_scores)), 4),
        },
        "scores_with_option_a": {
            "group_1_10_lengkap": {
                "min": round(float(np.min(g1)), 2),
                "mean": round(float(np.mean(g1)), 2),
                "max": round(float(np.max(g1)), 2),
            },
            "group_11_26_sebagian": {
                "min": round(float(np.min(g2)), 2),
                "mean": round(float(np.mean(g2)), 2),
                "max": round(float(np.max(g2)), 2),
            },
            "group_27_30_melenceng": {
                "min": round(float(np.min(g3)), 2),
                "mean": round(float(np.mean(g3)), 2),
                "max": round(float(np.max(g3)), 2),
            },
        },
        "similarity_copied_pair_bagas_dimas": {
            "semantic_cosine": round(copied_pair_sem, 4),
            "lexical_tfidf_cosine": round(copied_pair_lex, 4),
        },
        "similarity_other_pairs_max": {
            "semantic_max": round(float(np.max(pair_sems)), 4),
            "lexical_max": round(float(np.max(pair_lexs)), 4),
        },
    }
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
