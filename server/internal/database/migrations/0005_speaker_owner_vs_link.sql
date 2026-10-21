-- +goose Up
-- `speaker.user_id` was doing double duty: ownership AND "this voice belongs to
-- this person". Unlinking a speaker from a user therefore also removed it from
-- its owner's list. Split the two concepts.
ALTER TABLE speaker RENAME COLUMN user_id TO owner_id;
ALTER TABLE speaker ADD COLUMN linked_user_id UUID REFERENCES app_user(id) ON DELETE SET NULL;

ALTER INDEX IF EXISTS speaker_user_id_idx RENAME TO speaker_owner_id_idx;
CREATE INDEX speaker_linked_user_id_idx ON speaker (linked_user_id);

-- +goose Down
DROP INDEX IF EXISTS speaker_linked_user_id_idx;
ALTER INDEX IF EXISTS speaker_owner_id_idx RENAME TO speaker_user_id_idx;
ALTER TABLE speaker DROP COLUMN IF EXISTS linked_user_id;
ALTER TABLE speaker RENAME COLUMN owner_id TO user_id;
