-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE company(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(64),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE recording(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  path TEXT NOT NULL,
  extension VARCHAR(12) NOT NULL,
  filename TEXT NOT NULL,
  status TEXT NOT NULL
);

CREATE TABLE conversation(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  summary TEXT,
  length FLOAT8 NOT NULL,
  cost FLOAT8 NOT NULL,
  language VARCHAR(10) NOT NULL,
  language_confidence FLOAT8 NOT NULL,
  recording_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT fk_recording_id FOREIGN KEY (recording_id)
    REFERENCES recording(id)
);

CREATE TABLE speaker(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT,
  email VARCHAR(64),
  company_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT fk_company_id FOREIGN KEY (company_id)
    REFERENCES company(id)
);

CREATE TABLE speaker_embedding (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  speaker_id UUID NOT NULL REFERENCES speaker(id) ON DELETE CASCADE,
  embedding VECTOR(192) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON speaker_embedding USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

CREATE TABLE segment(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  text TEXT,
  start_time FLOAT8,
  end_time FLOAT8,
  duration FLOAT8,
  conversation_id UUID NOT NULL,
  speaker_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT fk_conversation_id FOREIGN KEY (conversation_id)
    REFERENCES conversation(id) ON DELETE CASCADE,
  CONSTRAINT fk_speaker_id FOREIGN KEY (speaker_id)
    REFERENCES speaker(id)
);


-- +goose Down
DROP TABLE IF EXISTS company;
DROP TABLE IF EXISTS conversation;
DROP TABLE IF EXISTS recording;
DROP TABLE IF EXISTS speaker;
DROP TABLE IF EXISTS segment;
