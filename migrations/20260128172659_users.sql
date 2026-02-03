-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  confirmed BOOLEAN NOT NULL DEFAULT FALSE,
  confirmation_token_hash VARCHAR(255) NULL DEFAULT NULL,
  confirmation_expires_at TIMESTAMP NULL DEFAULT NULL,
  password_reset_token_hash VARCHAR(255) NULL DEFAULT NULL,
  password_reset_expires_at TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
