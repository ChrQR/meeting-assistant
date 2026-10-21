-- +goose Up
-- Deleting a company should detach its speakers rather than fail.
ALTER TABLE speaker
  DROP CONSTRAINT fk_company_id;

ALTER TABLE speaker
  ADD CONSTRAINT fk_company_id FOREIGN KEY (company_id)
    REFERENCES company(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE speaker
  DROP CONSTRAINT fk_company_id;

ALTER TABLE speaker
  ADD CONSTRAINT fk_company_id FOREIGN KEY (company_id)
    REFERENCES company(id);
