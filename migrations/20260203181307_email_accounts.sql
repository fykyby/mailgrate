-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_accounts (
  id SERIAL PRIMARY KEY,
  sync_list_id INTEGER NOT NULL,
  login VARCHAR(255) NOT NULL,
  password VARCHAR(255) NOT NULL,
  FOREIGN KEY (sync_list_id) REFERENCES sync_lists (id) ON DELETE CASCADE,
  CONSTRAINT email_accounts_sync_list_login_unique UNIQUE (sync_list_id, login)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS email_accounts;

-- +goose StatementEnd
