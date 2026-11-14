import json
import traceback
from typing import Dict, Any
import numpy as np
from psycopg2.extras import Json
from app.db import get_db_connection
from app.preprocess import split_key_points
from app.scorer import EmbeddingEngine, score_single_submission
from app.similarity import detect_similar_pairs


def update_job_progress(
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


def process_scoring_job(r, payload: Dict[str, Any]):
    job_id = str(payload["job_id"])
    assignment_id = int(payload["assignment_id"])
    print(f"[SCORING] Memulai job={job_id} untuk assignment_id={assignment_id}", flush=True)

    conn = get_db_connection()
    try:
        update_job_progress(conn, r, job_id, "processing", "memecah teks", 10)

        with conn.cursor() as cur:
            # 1. Ambil pengaturan ambang AI
            cur.execute("SELECT t_high, t_low, s_sem, s_lex FROM ai_settings WHERE id = 1")
            ai_cfg = cur.fetchone()
            t_high = float(ai_cfg["t_high"]) if ai_cfg else 0.60
            t_low = float(ai_cfg["t_low"]) if ai_cfg else 0.45
            s_sem = float(ai_cfg["s_sem"]) if ai_cfg else 0.92
            s_lex = float(ai_cfg["s_lex"]) if ai_cfg else 0.60

            # 2. Ambil data tugas
            cur.execute(
                """
                SELECT id, user_id, title, max_score, model_type, custom_model_id
                FROM assignments
                WHERE id = %s
                """,
                (assignment_id,),
            )
            assignment = cur.fetchone()
            if not assignment:
                raise RuntimeError(f"Tugas dengan ID {assignment_id} tidak ditemukan.")

            max_score = float(assignment["max_score"])
            model_type = assignment["model_type"]
            custom_model_id = assignment["custom_model_id"]

            # 3. Ambil kriteria rubrik
            cur.execute(
                """
                SELECT id, name, weight, answer_key, sort_order
                FROM rubric_criteria
                WHERE assignment_id = %s
                ORDER BY sort_order ASC, id ASC
                """,
                (assignment_id,),
            )
            criteria_rows = cur.fetchall()
            if not criteria_rows:
                raise RuntimeError("Tugas belum memiliki kriteria rubrik.")

            # 4. Ambil seluruh jawaban siswa pada tugas ini
            cur.execute(
                """
                SELECT id, student_name, answer_text, revised_score, quota_charged
                FROM submissions
                WHERE assignment_id = %s
                ORDER BY id ASC
                """,
                (assignment_id,),
            )
            submissions = cur.fetchall()

        if not submissions:
            update_job_progress(
                conn,
                r,
                job_id,
                "completed",
                "selesai",
                100,
                metadata={"total_scored": 0, "similarity_flags": 0},
            )
            return

        engine = EmbeddingEngine.get_instance()

        # Pre-encode poin kunci untuk setiap kriteria rubrik
        criteria_data = []
        all_key_points = []
        for c_row in criteria_rows:
            pts = split_key_points(c_row["answer_key"])
            vecs = engine.encode(pts) if pts else np.zeros((0, 768), dtype=np.float32)
            criteria_data.append(
                {
                    "id": c_row["id"],
                    "name": c_row["name"],
                    "weight": float(c_row["weight"]),
                    "key_points": pts,
                    "key_vecs": vecs,
                }
            )
            all_key_points.extend(pts)

        # Cek apakah memakai model custom yang memiliki versi aktif (untuk Fase 4)
        custom_predictor = None
        if model_type == "custom" and custom_model_id:
            try:
                from app.trainer import load_active_custom_predictor
                custom_predictor = load_active_custom_predictor(conn, int(custom_model_id))
            except Exception as exc:
                print(f"[SCORING] Info: fallback ke model bawaan ({exc})", flush=True)
                custom_predictor = None

        update_job_progress(conn, r, job_id, "processing", "menilai", 25)

        total_subs = len(submissions)
        doc_vecs = []
        newly_charged_ids = []

        for idx, sub in enumerate(submissions):
            sub_id = sub["id"]
            ans_text = sub["answer_text"]

            res, doc_vec = score_single_submission(
                engine=engine,
                answer_text=ans_text,
                criteria_data=criteria_data,
                t_high=t_high,
                t_low=t_low,
                max_score=max_score,
            )
            doc_vecs.append(doc_vec)

            predicted_score = res["predicted_score"]
            if custom_predictor is not None:
                pred_norm = custom_predictor.predict_norm(
                    answer_text=ans_text,
                    raw_point_scores=res["raw_point_scores"],
                    crit_slices=res["crit_slices"],
                    crit_weights=res["crit_weights"],
                    all_key_points=all_key_points,
                )
                predicted_score = round(float(np.clip(pred_norm, 0.0, 1.0)) * max_score, 2)

            # Bila guru sudah pernah memberi revised_score, pertahankan final_score = revised_score
            revised_score = sub["revised_score"]
            final_score = float(revised_score) if revised_score is not None else predicted_score

            with conn.cursor() as cur:
                cur.execute(
                    """
                    UPDATE submissions
                    SET predicted_score = %s,
                        final_score = %s,
                        feedback_text = %s,
                        status = 'scored',
                        quota_charged = TRUE,
                        updated_at = NOW()
                    WHERE id = %s
                    """,
                    (predicted_score, final_score, res["feedback_text"], sub_id),
                )

                for c_score in res["criteria_scores"]:
                    cur.execute(
                        """
                        INSERT INTO submission_criteria_scores (
                            submission_id, criterion_id, coverage_score, status,
                            matched_snippet, missing_key_points, feedback_sentence
                        )
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                        ON CONFLICT (submission_id, criterion_id) DO UPDATE SET
                            coverage_score = EXCLUDED.coverage_score,
                            status = EXCLUDED.status,
                            matched_snippet = EXCLUDED.matched_snippet,
                            missing_key_points = EXCLUDED.missing_key_points,
                            feedback_sentence = EXCLUDED.feedback_sentence
                        """,
                        (
                            sub_id,
                            c_score["criterion_id"],
                            c_score["coverage_score"],
                            c_score["status"],
                            c_score["matched_snippet"],
                            Json(c_score["missing_key_points"]),
                            c_score["feedback_sentence"],
                        ),
                    )

            if not sub["quota_charged"]:
                newly_charged_ids.append(sub_id)

            conn.commit()

            # Update progres 25% -> 80%
            pct = 25 + int(((idx + 1) / total_subs) * 55)
            update_job_progress(
                conn,
                r,
                job_id,
                "processing",
                "menilai",
                pct,
                metadata={"scored_count": idx + 1, "total_count": total_subs},
            )

        # Potong kuota hanya untuk jawaban siswa yang baru pertama kali dinilai
        if newly_charged_ids:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    UPDATE subscriptions
                    SET quota_used = quota_used + %s,
                        updated_at = NOW()
                    WHERE user_id = %s
                    """,
                    (len(newly_charged_ids), assignment["user_id"]),
                )
            conn.commit()

        # Tahap 4: Cek kemiripan antar siswa dalam tugas ini
        update_job_progress(conn, r, job_id, "processing", "cek kemiripan", 88)
        doc_matrix = np.vstack(doc_vecs) if doc_vecs else np.zeros((0, 768), dtype=np.float32)
        flagged_pairs = detect_similar_pairs(
            submissions=submissions,
            doc_embeddings=doc_matrix,
            s_sem=s_sem,
            s_lex=s_lex,
        )

        with conn.cursor() as cur:
            cur.execute("DELETE FROM similarity_flags WHERE assignment_id = %s", (assignment_id,))
            for fp in flagged_pairs:
                cur.execute(
                    """
                    INSERT INTO similarity_flags (
                        assignment_id, submission_a_id, submission_b_id,
                        semantic_score, lexical_score, matched_excerpt
                    )
                    VALUES (%s, %s, %s, %s, %s, %s)
                    """,
                    (
                        assignment_id,
                        fp["submission_a_id"],
                        fp["submission_b_id"],
                        fp["semantic_score"],
                        fp["lexical_score"],
                        fp["matched_excerpt"],
                    ),
                )
        conn.commit()

        update_job_progress(
            conn,
            r,
            job_id,
            "completed",
            "selesai",
            100,
            metadata={
                "total_scored": total_subs,
                "newly_charged": len(newly_charged_ids),
                "similarity_flags": len(flagged_pairs),
            },
        )
        print(
            f"[SCORING] Selesai job={job_id}: {total_subs} jawaban dinilai, "
            f"{len(flagged_pairs)} pasangan mirip terdeteksi.",
            flush=True,
        )
    except Exception as exc:
        traceback.print_exc()
        try:
            conn.rollback()
            update_job_progress(conn, r, job_id, "failed", "gagal", 100, error_message=str(exc))
        except Exception:
            pass
    finally:
        conn.close()

