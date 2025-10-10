-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;

-- Обновляем все существующие аккаунты как активные (на всякий случай)
UPDATE accounts SET is_active = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE accounts
DROP COLUMN IF EXISTS is_active;
-- +goose StatementEnd