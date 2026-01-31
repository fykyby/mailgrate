-- +goose Up
-- +goose StatementBegin
CREATE TABLE jobs (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL,
  type VARCHAR(255) NOT NULL,
  status VARCHAR(255) NOT NULL,
  payload JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  started_at TIMESTAMP DEFAULT NULL,
  finished_at TIMESTAMP DEFAULT NULL,
  FOREIGN KEY (user_id) REFERENCES users (id)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE jobs;

-- +goose StatementEnd
