-- +goose Up
-- +goose StatementBegin
CREATE TABLE sync_lists (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL,
  name VARCHAR(255) NOT NULL,
  source_host VARCHAR(255) NOT NULL,
  source_port INT NOT NULL,
  destination_host VARCHAR(255) NOT NULL,
  destination_port INT NOT NULL,
  status VARCHAR(255) NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users (id),
  CONSTRAINT sync_lists_user_name_unique UNIQUE (user_id, name)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sync_lists;

-- +goose StatementEnd
