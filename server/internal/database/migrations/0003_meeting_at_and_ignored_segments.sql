-- +goose Up
-- The date the meeting actually took place. Nullable: older rows and uploads
-- without an explicit date fall back to created_at in the UI.
ALTER TABLE conversation
  ADD COLUMN meeting_at TIMESTAMPTZ;

-- Segments whose speaker cannot be identified (typically very short utterances)
-- can be ignored. Ignored segments stay in the transcript for reference but do
-- not count towards the "unknown speakers" warning.
ALTER TABLE segment
  ADD COLUMN ignored BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE segment DROP COLUMN IF EXISTS ignored;
ALTER TABLE conversation DROP COLUMN IF EXISTS meeting_at;
