-- +goose Up
-- +goose StatementBegin
ALTER TABLE note_files
ADD COLUMN ext VARCHAR(10);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE note_files
DROP COLUMN ext;
-- +goose StatementEnd
