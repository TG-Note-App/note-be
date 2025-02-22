-- +goose Up
-- +goose StatementBegin
ALTER TABLE note_files
ADD COLUMN "size" BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE note_files
DROP COLUMN "size";
-- +goose StatementEnd
