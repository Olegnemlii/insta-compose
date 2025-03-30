-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL  DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    is_confirmed BOOLEAN DEFAULT false
);

CREATE TABLE tokens (
    id SERIAL PRIMARY KEY,
    access_token VARCHAR NOT NULL,
    refresh_token VARCHAR NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id)
);

CREATE TABLE codes_signatures(
    code VARCHAR(255) NOT NULL,
    signature VARCHAR NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    is_used BOOLEAN DEFAULT false,
    expires_at TIMESTAMPTZ NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS codes_signatures;

DROP TABLE IF EXISTS tokens;

DROP TABLE IF EXISTS users;
-- +goose StatementEnd