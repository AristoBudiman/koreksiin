-- 002_model_version_mode.sql: Penyesuaian kolom custom_model_versions & payment_notification_logs

ALTER TABLE custom_model_versions
ADD COLUMN IF NOT EXISTS training_mode VARCHAR(30) NOT NULL DEFAULT 'calibration';

ALTER TABLE custom_model_versions
ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE payment_notification_logs
ADD COLUMN IF NOT EXISTS signature_valid BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE payment_notification_logs
ADD COLUMN IF NOT EXISTS processed_note TEXT NOT NULL DEFAULT '';
