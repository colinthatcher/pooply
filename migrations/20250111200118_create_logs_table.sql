-- +goose Up
-- +goose StatementBegin
CREATE TABLE logs (
  id UUID PRIMARY KEY,
  username VARCHAR(255) NOT NULL,
  started TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  ended TIMESTAMP DEFAULT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE logs;
-- +goose StatementEnd
