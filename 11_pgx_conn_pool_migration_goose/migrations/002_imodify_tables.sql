-- +goose Up
-- +goose StatementBegin
delete where id = 10
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
insert id 10
-- +goose StatementEnd