import os


class Config:
    WORKER_ROLE = os.getenv("WORKER_ROLE", "scoring")
    WORKER_QUEUE = os.getenv("WORKER_QUEUE", "queue:scoring")
    DATABASE_URL = os.getenv(
        "DATABASE_URL",
        "postgres://koreksiin:koreksiin_secret_db_pass@postgres:5432/koreksiin_db?sslmode=disable",
    )
    REDIS_URL = os.getenv("REDIS_URL", "redis://redis:6379/0")
    BASE_MODEL_NAME = os.getenv("BASE_MODEL_NAME", "LazarusNLP/all-indobert-base-v4")
    HF_HOME = os.getenv("HF_HOME", "/hf_cache")
    STORAGE_DIR = os.getenv("STORAGE_DIR", "/storage")
    MIN_TRAIN_SAMPLES = int(os.getenv("MIN_TRAIN_SAMPLES", "20"))

