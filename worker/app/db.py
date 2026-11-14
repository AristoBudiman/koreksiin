import time
import psycopg2
from psycopg2.extras import RealDictCursor
from app.config import Config


def get_db_connection(retries: int = 10, delay: float = 2.0):
    last_err = None
    for attempt in range(1, retries + 1):
        try:
            conn = psycopg2.connect(Config.DATABASE_URL, cursor_factory=RealDictCursor)
            return conn
        except Exception as exc:
            last_err = exc
            print(f"[Worker DB] Menunggu PostgreSQL siap ({attempt}/{retries}): {exc}", flush=True)
            time.sleep(delay)
    raise RuntimeError(f"Gagal terhubung ke PostgreSQL: {last_err}")

