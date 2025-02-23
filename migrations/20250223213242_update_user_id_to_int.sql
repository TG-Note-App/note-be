-- +goose Up
-- +goose StatementBegin
UPDATE notes SET user_id = '0' WHERE user_id = '';
ALTER TABLE notes
ALTER COLUMN user_id TYPE INT USING NULLIF(user_id, '')::integer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notes
ALTER COLUMN user_id TYPE TEXT;
-- +goose StatementEnd
