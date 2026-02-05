-- +goose Up
-- +goose StatementBegin
CREATE TABLE mailboxes (
  id SERIAL PRIMARY KEY,
  sync_list_id INTEGER NOT NULL,
  src_user VARCHAR(255) NOT NULL,
  src_password_hash VARCHAR(255) NOT NULL,
  dst_user VARCHAR(255) NOT NULL,
  dst_password_hash VARCHAR(255) NOT NULL,
  folder_last_uid JSONB NOT NULL,
  folder_uid_validity JSONB NOT NULL,
  FOREIGN KEY (sync_list_id) REFERENCES sync_lists (id) ON DELETE CASCADE,
  CONSTRAINT mailboxes_sync_list_user_unique UNIQUE (sync_list_id, src_user, dst_user)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS mailboxes;

-- +goose StatementEnd
