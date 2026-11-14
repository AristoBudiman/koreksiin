-- 001_init.sql: Skema lengkap database Koreksi.in

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Tabel users (Admin & Guru)
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(180) NOT NULL UNIQUE,
    institution VARCHAR(200) NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'guru')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Tabel plans (Paket Langganan & Add-on Kuota)
CREATE TABLE IF NOT EXISTS plans (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(40) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_idr INTEGER NOT NULL DEFAULT 0,
    quota_essays INTEGER NOT NULL DEFAULT 0,
    duration_days INTEGER NOT NULL DEFAULT 0,
    is_addon BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Tabel subscriptions (Langganan & Kuota Guru)
CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    quota_total INTEGER NOT NULL DEFAULT 0,
    quota_used INTEGER NOT NULL DEFAULT 0,
    quota_reset_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Tabel ai_settings (Singleton pengaturan ambang AI)
CREATE TABLE IF NOT EXISTS ai_settings (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    t_high DOUBLE PRECISION NOT NULL DEFAULT 0.60,
    t_low DOUBLE PRECISION NOT NULL DEFAULT 0.45,
    s_sem DOUBLE PRECISION NOT NULL DEFAULT 0.92,
    s_lex DOUBLE PRECISION NOT NULL DEFAULT 0.60,
    min_train_samples INTEGER NOT NULL DEFAULT 20,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Tabel custom_models (Model Custom milik Guru)
CREATE TABLE IF NOT EXISTS custom_models (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active_version_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Tabel custom_model_versions (Versi hasil training Kalibrasi / LoRA)
CREATE TABLE IF NOT EXISTS custom_model_versions (
    id BIGSERIAL PRIMARY KEY,
    model_id BIGINT NOT NULL REFERENCES custom_models(id) ON DELETE CASCADE,
    version_num INTEGER NOT NULL,
    training_mode VARCHAR(30) NOT NULL DEFAULT 'calibration',
    storage_path TEXT NOT NULL,
    sample_count INTEGER NOT NULL DEFAULT 0,
    train_count INTEGER NOT NULL DEFAULT 0,
    test_count INTEGER NOT NULL DEFAULT 0,
    mae_custom DOUBLE PRECISION NOT NULL DEFAULT 0,
    pearson_custom DOUBLE PRECISION NOT NULL DEFAULT 0,
    mae_base DOUBLE PRECISION NOT NULL DEFAULT 0,
    pearson_base DOUBLE PRECISION NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (model_id, version_num)
);

-- Tambahkan FK active_version_id setelah tabel custom_model_versions dibuat
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_custom_models_active_version'
    ) THEN
        ALTER TABLE custom_models
        ADD CONSTRAINT fk_custom_models_active_version
        FOREIGN KEY (active_version_id) REFERENCES custom_model_versions(id) ON DELETE SET NULL;
    END IF;
END $$;

-- 7. Tabel assignments (Tugas Penilaian Esai)
CREATE TABLE IF NOT EXISTS assignments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    subject VARCHAR(120) NOT NULL,
    question_text TEXT NOT NULL,
    max_score DOUBLE PRECISION NOT NULL DEFAULT 100,
    model_type VARCHAR(20) NOT NULL DEFAULT 'base' CHECK (model_type IN ('base', 'custom')),
    custom_model_id BIGINT REFERENCES custom_models(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 8. Tabel custom_model_assignments (Relasi Model Custom <-> Tugas sumber data latih)
CREATE TABLE IF NOT EXISTS custom_model_assignments (
    model_id BIGINT NOT NULL REFERENCES custom_models(id) ON DELETE CASCADE,
    assignment_id BIGINT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    PRIMARY KEY (model_id, assignment_id)
);

-- 9. Tabel rubric_criteria (Kriteria Rubrik per Tugas)
CREATE TABLE IF NOT EXISTS rubric_criteria (
    id BIGSERIAL PRIMARY KEY,
    assignment_id BIGINT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    weight DOUBLE PRECISION NOT NULL DEFAULT 1,
    answer_key TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

-- 10. Tabel submissions (Jawaban Siswa)
CREATE TABLE IF NOT EXISTS submissions (
    id BIGSERIAL PRIMARY KEY,
    assignment_id BIGINT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    student_name VARCHAR(150) NOT NULL,
    answer_text TEXT NOT NULL,
    predicted_score DOUBLE PRECISION,
    revised_score DOUBLE PRECISION,
    final_score DOUBLE PRECISION,
    feedback_text TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'scored')),
    quota_charged BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 11. Tabel submission_criteria_scores (Skor & Status per Kriteria untuk setiap Siswa)
CREATE TABLE IF NOT EXISTS submission_criteria_scores (
    id BIGSERIAL PRIMARY KEY,
    submission_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    criterion_id BIGINT NOT NULL REFERENCES rubric_criteria(id) ON DELETE CASCADE,
    coverage_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL CHECK (status IN ('terpenuhi', 'sebagian', 'tidak_ada')),
    matched_snippet TEXT NOT NULL DEFAULT '',
    missing_key_points JSONB NOT NULL DEFAULT '[]'::jsonb,
    feedback_sentence TEXT NOT NULL DEFAULT '',
    UNIQUE (submission_id, criterion_id)
);

-- 12. Tabel similarity_flags (Pasangan Jawaban Siswa yang Terindikasi Mirip)
CREATE TABLE IF NOT EXISTS similarity_flags (
    id BIGSERIAL PRIMARY KEY,
    assignment_id BIGINT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    submission_a_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    submission_b_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    semantic_score DOUBLE PRECISION NOT NULL,
    lexical_score DOUBLE PRECISION NOT NULL,
    matched_excerpt TEXT NOT NULL DEFAULT ''
);

-- 13. Tabel training_samples (Data Latih yang Terhubung ke Tugas)
CREATE TABLE IF NOT EXISTS training_samples (
    id BIGSERIAL PRIMARY KEY,
    assignment_id BIGINT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    submission_id BIGINT UNIQUE REFERENCES submissions(id) ON DELETE SET NULL,
    student_name VARCHAR(150) NOT NULL,
    answer_text TEXT NOT NULL,
    target_score DOUBLE PRECISION NOT NULL,
    source VARCHAR(20) NOT NULL CHECK (source IN ('revision', 'excel')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 14. Tabel jobs (Status Job Penilaian & Training)
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('scoring', 'training')),
    queue_name VARCHAR(40) NOT NULL,
    reference_id BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
    stage VARCHAR(100) NOT NULL DEFAULT 'antri',
    progress INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 15. Tabel payment_transactions (Transaksi Midtrans Sandbox)
CREATE TABLE IF NOT EXISTS payment_transactions (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(100) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    gross_amount INTEGER NOT NULL,
    snap_token TEXT NOT NULL DEFAULT '',
    snap_redirect_url TEXT NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    payment_type VARCHAR(50) NOT NULL DEFAULT '',
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 16. Tabel payment_notification_logs (Log Webhook Midtrans)
CREATE TABLE IF NOT EXISTS payment_notification_logs (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(100) NOT NULL,
    status_code VARCHAR(20) NOT NULL,
    transaction_status VARCHAR(40) NOT NULL,
    signature_key TEXT NOT NULL,
    signature_valid BOOLEAN NOT NULL DEFAULT FALSE,
    processed_note TEXT NOT NULL DEFAULT '',
    raw_payload JSONB NOT NULL,
    processed_idempotently BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assignments_user_id ON assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_assignment_id ON submissions(assignment_id);
CREATE INDEX IF NOT EXISTS idx_training_samples_assignment_id ON training_samples(assignment_id);
CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_jobs_reference ON jobs(type, reference_id);

