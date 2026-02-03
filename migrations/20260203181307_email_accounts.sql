-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_accounts (
  id SERIAL PRIMARY KEY,
  sync_list_id INTEGER NOT NULL,
  login VARCHAR(255) NOT NULL,
  password VARCHAR(255) NOT NULL,
  FOREIGN KEY (sync_list_id) REFERENCES sync_lists (id),
  CONSTRAINT email_accounts_sync_list_login_unique UNIQUE (sync_list_id, login)
);

CREATE TABLE email_accounts_jobs (
  id SERIAL PRIMARY KEY,
  email_account_id INTEGER NOT NULL,
  job_id INTEGER NOT NULL,
  FOREIGN KEY (email_account_id) REFERENCES email_accounts (id),
  FOREIGN KEY (job_id) REFERENCES jobs (id)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS email_accounts_jobs;

DROP TABLE IF EXISTS email_accounts;

-- +goose StatementEnd
