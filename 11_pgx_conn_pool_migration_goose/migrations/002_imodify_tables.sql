-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_logs (
                            id SERIAL PRIMARY KEY,
                            account_id INTEGER REFERENCES accounts(id) ON DELETE SET NULL,
                            action TEXT NOT NULL CHECK (action IN ('login', 'logout', 'update_profile', 'delete_account')),
    details JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_account_id ON audit_logs (account_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_account_id;
DROP TABLE IF EXISTS audit_logs;
-- +goose StatementEnd