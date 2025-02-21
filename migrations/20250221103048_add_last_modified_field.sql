-- +goose Up
-- +goose StatementBegin
ALTER TABLE notes ADD COLUMN last_modified TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
UPDATE notes SET last_modified = CURRENT_TIMESTAMP;
ALTER TABLE notes ALTER COLUMN last_modified SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notes DROP COLUMN last_modified;
-- +goose StatementEnd
