import json
import os
import signal
import sys
import time
import redis
from app.config import Config
from app.db import get_db_connection

running = True


def handle_sigterm(signum, frame):
    global running
    print(f"[{Config.WORKER_ROLE}] Menerima sinyal berhenti, menyelesaikan siklus...", flush=True)
    running = False


signal.signal(signal.SIGTERM, handle_sigterm)
signal.signal(signal.SIGINT, handle_sigterm)


def main():
    os.makedirs(Config.STORAGE_DIR, exist_ok=True)
    os.makedirs(Config.HF_HOME, exist_ok=True)

    conn = get_db_connection()
    conn.close()

    r = redis.from_url(Config.REDIS_URL, decode_responses=True)
    for attempt in range(1, 15):
        try:
            r.ping()
            break
        except Exception as exc:
            print(f"[{Config.WORKER_ROLE}] Menunggu Redis ({attempt}/15): {exc}", flush=True)
            time.sleep(1.5)

    # Simpan heartbeat awal di Redis
    r.set(f"worker:heartbeat:{Config.WORKER_ROLE}", str(int(time.time())))
    print(
        f"[{Config.WORKER_ROLE.upper()}] Siap mendengarkan antrian '{Config.WORKER_QUEUE}' "
        f"(model dasar: {Config.BASE_MODEL_NAME})",
        flush=True,
    )

    while running:
        try:
            r.set(f"worker:heartbeat:{Config.WORKER_ROLE}", str(int(time.time())), ex=30)
            item = r.blpop(Config.WORKER_QUEUE, timeout=5)
            if not item:
                continue
            _, raw_payload = item
            payload = json.loads(raw_payload)
            print(f"[{Config.WORKER_ROLE.upper()}] Menerima job: {payload}", flush=True)
        except Exception as exc:
            print(f"[{Config.WORKER_ROLE.upper()}] Error pada loop utama: {exc}", flush=True)
            time.sleep(2)

    print(f"[{Config.WORKER_ROLE.upper()}] Berhenti dengan bersih.", flush=True)
    sys.exit(0)


if __name__ == "__main__":
    main()

