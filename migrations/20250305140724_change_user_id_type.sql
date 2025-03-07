-- +goose Up
-- Change user_id column type from int to bigint in notes table
ALTER TABLE notes ALTER COLUMN user_id TYPE bigint;

-- +goose Down
-- Revert user_id column type from bigint to int in notes table
ALTER TABLE notes ALTER COLUMN user_id TYPE int;
