-- +goose Up
-- +goose StatementBegin
CREATE TABLE jobs (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL,
  related_table VARCHAR(255) DEFAULT NULL,
  related_id INT DEFAULT NULL,
  type VARCHAR(255) NOT NULL,
  status VARCHAR(255) NOT NULL,
  payload JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  started_at TIMESTAMP DEFAULT NULL,
  finished_at TIMESTAMP DEFAULT NULL,
  error VARCHAR(255) DEFAULT NULL,
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE FUNCTION notify_job_update () RETURNS TRIGGER AS $$
BEGIN
  IF NEW.status IN ('pending') THEN
    PERFORM pg_notify('jobs:updated', NEW.id::text);
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER jobs_after_update_trigger
AFTER INSERT
OR
UPDATE ON jobs FOR EACH ROW
EXECUTE PROCEDURE notify_job_update ();

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS jobs_after_update_trigger ON jobs;

DROP FUNCTION IF EXISTS notify_job_update ();

DROP TABLE IF EXISTS jobs;

-- +goose StatementEnd
