-- +goose Up
-- +goose StatementBegin
ALTER TABLE notes ADD COLUMN is_pin BOOLEAN DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notes DROP COLUMN is_pin;
-- +goose StatementEnd
