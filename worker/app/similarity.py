from typing import List, Dict, Any
import numpy as np
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity


def compute_lexical_similarity_matrix(texts: List[str]) -> np.ndarray:
    """
    Menghitung matriks kemiripan leksikal antar teks jawaban siswa
    menggunakan cosine TF-IDF word n-gram (1, 2).
    """
    n = len(texts)
    if n == 0:
        return np.zeros((0, 0), dtype=np.float32)
    if n == 1:
        return np.ones((1, 1), dtype=np.float32)

    vectorizer = TfidfVectorizer(ngram_range=(1, 2), lowercase=True)
    try:
        tfidf_matrix = vectorizer.fit_transform(texts)
        sim_matrix = cosine_similarity(tfidf_matrix)
        return sim_matrix.astype(np.float32)
    except ValueError:
        return np.zeros((n, n), dtype=np.float32)


def detect_similar_pairs(
    submissions: List[Dict[str, Any]],
    doc_embeddings: np.ndarray,
    s_sem: float,
    s_lex: float,
) -> List[Dict[str, Any]]:
    """
    Mendeteksi pasangan jawaban siswa dalam satu tugas yang melampaui
    S_SEM (semantik cosine) DAN S_LEX (leksikal TF-IDF cosine).
    """
    n = len(submissions)
    if n < 2:
        return []

    texts = [s["answer_text"] for s in submissions]
    sem_matrix = np.matmul(doc_embeddings, doc_embeddings.T)
    lex_matrix = compute_lexical_similarity_matrix(texts)

    flagged = []
    for i in range(n):
        for j in range(i + 1, n):
            sem_score = float(sem_matrix[i, j])
            lex_score = float(lex_matrix[i, j])
            if sem_score >= s_sem and lex_score >= s_lex:
                excerpt = texts[i][:180].strip()
                if len(texts[i]) > 180:
                    excerpt += "..."
                flagged.append(
                    {
                        "submission_a_id": submissions[i]["id"],
                        "submission_b_id": submissions[j]["id"],
                        "semantic_score": round(sem_score, 4),
                        "lexical_score": round(lex_score, 4),
                        "matched_excerpt": excerpt,
                    }
                )
    return flagged

