-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE company(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(64),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE conversation(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  summary TEXT,
  length FLOAT8 NOT NULL,
  cost FLOAT8 NOT NULL,
  language VARCHAR(10) NOT NULL,
  language_confidence FLOAT8 NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE speaker(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT,
  email VARCHAR(64),
  company_id UUID,
  voice_embedding VECTOR(512),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT fk_company_id FOREIGN KEY (company_id)
    REFERENCES company(id)
);

CREATE INDEX ON speaker USING hnsw (voice_embedding vector_cosine_ops);


CREATE TABLE segment(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  text TEXT,
  cost FLOAT8,
  start_time FLOAT8,
  end_time FLOAT8,
  confidence FLOAT8,
  conversation_id UUID NOT NULL,
  speaker_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT fk_conversation_id FOREIGN KEY (conversation_id)
    REFERENCES conversation(id),
  CONSTRAINT fk_speaker_id FOREIGN KEY (speaker_id)
    REFERENCES speaker(id)
);


-- +goose Down
DROP TABLE IF EXISTS company;
DROP TABLE IF EXISTS conversation;
DROP TABLE IF EXISTS speaker;
DROP TABLE IF EXISTS segment;
