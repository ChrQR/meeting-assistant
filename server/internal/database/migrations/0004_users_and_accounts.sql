-- +goose Up
-- DESTRUCTIVE: this migration introduces multi-user ownership. All existing
-- meetings, speakers and companies belong to the single hardcoded development
-- user and cannot be attributed to a real account, so they are removed.
TRUNCATE conversation, segment, recording, speaker_embedding, speaker, company CASCADE;

CREATE TABLE app_user (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  name TEXT,
  avatar_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT app_user_email_unique UNIQUE (email)
);

-- One row per external identity provider linked to a user. Keeping provider
-- identities in their own table means adding another SSO provider later does
-- not change the user entity.
CREATE TABLE account (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  provider_account_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT account_provider_identity_unique UNIQUE (provider, provider_account_id)
);

CREATE INDEX account_user_id_idx ON account (user_id);

-- Recordings belong to the user that uploaded them.
ALTER TABLE recording
  ADD COLUMN user_id UUID REFERENCES app_user(id) ON DELETE CASCADE;

-- A speaker can be linked to an application user ("this voice is this person").
ALTER TABLE speaker
  ADD COLUMN user_id UUID REFERENCES app_user(id) ON DELETE SET NULL;

-- Companies are personal to the user that created them.
ALTER TABLE company
  ADD COLUMN user_id UUID REFERENCES app_user(id) ON DELETE CASCADE;

CREATE INDEX recording_user_id_idx ON recording (user_id);
CREATE INDEX speaker_user_id_idx ON speaker (user_id);
CREATE INDEX company_user_id_idx ON company (user_id);

-- Conversations are owned through their recording; speakers are personal to the
-- user that created them (voice embeddings must not leak across accounts).

-- +goose Down
ALTER TABLE company DROP COLUMN IF EXISTS user_id;
ALTER TABLE speaker DROP COLUMN IF EXISTS user_id;
ALTER TABLE recording DROP COLUMN IF EXISTS user_id;
DROP TABLE IF EXISTS account;
DROP TABLE IF EXISTS app_user;
