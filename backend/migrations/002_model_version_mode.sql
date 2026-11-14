-- 002_model_version_mode.sql: Tambahan kolom mode pelatihan & metadata CV pada custom_model_versions

ALTER TABLE custom_model_versions
ADD COLUMN IF NOT EXISTS training_mode VARCHAR(30) NOT NULL DEFAULT 'calibration';

ALTER TABLE custom_model_versions
ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

