-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE images (
    id                SERIAL PRIMARY KEY,
    user_id           INTEGER NOT NULL,
    item_id           INTEGER NOT NULL,
    sku               VARCHAR(64) DEFAULT NULL,
    context           VARCHAR(64) NOT NULL,
    description       VARCHAR(255) DEFAULT NULL,
    width             SMALLINT NOT NULL,
    height            SMALLINT NOT NULL,
    project           VARCHAR(64) NOT NULL,
    size              INTEGER NOT NULL,
    key               VARCHAR(255) NOT NULL,
    webp_key          VARCHAR(255) DEFAULT NULL,
    mime_type         VARCHAR(10) NOT NULL,
    is_deleted        BOOLEAN DEFAULT FALSE,
    order_index       SMALLINT NOT NULL,
    created_timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project, user_id, key)
);

CREATE INDEX idx_images_lookup ON images (project, user_id, key);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
