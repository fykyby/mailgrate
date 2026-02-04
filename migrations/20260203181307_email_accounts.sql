-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_accounts (
  id SERIAL PRIMARY KEY,
  sync_list_id INTEGER NOT NULL,
  src_user VARCHAR(255) NOT NULL,
  src_password_hash VARCHAR(255) NOT NULL,
  dst_user VARCHAR(255) NOT NULL,
  dst_password_hash VARCHAR(255) NOT NULL,
  FOREIGN KEY (sync_list_id) REFERENCES sync_lists (id) ON DELETE CASCADE,
  CONSTRAINT email_accounts_sync_list_user_unique UNIQUE (sync_list_id, src_user, dst_user)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS email_accounts;

-- +goose StatementEnd
