-- +goose Up
-- +goose StatementBegin
CREATE TABLE sync_lists (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL,
  name VARCHAR(255) NOT NULL,
  src_host VARCHAR(255) NOT NULL,
  src_port INT NOT NULL,
  dst_host VARCHAR(255) NOT NULL,
  dst_port INT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT sync_lists_user_name_unique UNIQUE (user_id, name)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sync_lists;

-- +goose StatementEnd
