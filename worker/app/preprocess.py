import re
from typing import List


def split_sentences(text: str) -> List[str]:
    """Memecah teks menjadi daftar kalimat bersih."""
    if not text:
        return []
    cleaned = re.sub(r"\r\n|\r", "\n", text.strip())
    # Pecah berdasarkan baris baru atau tanda baca akhir kalimat (. ! ? ;)
    raw_parts = re.split(r"(?<=[.!?;])\s+|\n+", cleaned)
    sentences = []
    for part in raw_parts:
        s = re.sub(r"^[\s\-*•\d.)]+", "", part).strip()
        if s:
            sentences.append(s)
    return sentences


def split_key_points(answer_key: str) -> List[str]:
    """Memecah kunci jawaban tiap kriteria per kalimat/baris menjadi poin kunci."""
    pts = split_sentences(answer_key)
    if not pts and answer_key.strip():
        return [answer_key.strip()]
    return pts


def build_answer_windows(answer_text: str, tokenizer=None, max_tokens: int = 512) -> List[str]:
    """
    Memecah jawaban siswa per kalimat, lalu membentuk jendela 1-3 kalimat
    yang tumpang tindih. Potongan yang melebihi 512 token dipecah lagi.
    """
    sentences = split_sentences(answer_text)
    if not sentences:
        if answer_text and answer_text.strip():
            sentences = [answer_text.strip()]
        else:
            return []

    raw_windows = []
    n = len(sentences)
    for w_size in (1, 2, 3):
        if n < w_size:
            continue
        for i in range(n - w_size + 1):
            window_text = " ".join(sentences[i : i + w_size]).strip()
            if window_text and window_text not in raw_windows:
                raw_windows.append(window_text)

    if not raw_windows:
        raw_windows = [" ".join(sentences).strip()]

    if tokenizer is None:
        return raw_windows

    final_windows = []
    safe_token_limit = max_tokens - 4  # ruang untuk [CLS] dan [SEP]
    for win in raw_windows:
        token_ids = tokenizer.encode(win, add_special_tokens=False)
        if len(token_ids) <= safe_token_limit:
            if win not in final_windows:
                final_windows.append(win)
        else:
            # Pecah potongan yang melebihi 512 token dengan stride bertumpuk
            stride = max(64, safe_token_limit // 2)
            for start in range(0, len(token_ids), stride):
                sub_ids = token_ids[start : start + safe_token_limit]
                sub_text = tokenizer.decode(sub_ids, skip_special_tokens=True).strip()
                if sub_text and sub_text not in final_windows:
                    final_windows.append(sub_text)
                if start + safe_token_limit >= len(token_ids):
                    break

    return final_windows

