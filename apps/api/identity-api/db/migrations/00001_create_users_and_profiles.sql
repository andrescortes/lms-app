-- +goose Up
-- Users and profiles for the Identity service.
-- STRICT tables; role constrained to the known set; normalized email is unique;
-- a profile belongs to exactly one user and is removed with that user.

CREATE TABLE users (
    id            TEXT    PRIMARY KEY NOT NULL,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    display_name  TEXT    NOT NULL,
    role          TEXT    NOT NULL CHECK (role IN ('STUDENT','TEACHER','ADMIN','REVIEWER')),
    cefr_level    TEXT    NULL,
    created_at    TEXT    NOT NULL
) STRICT;

CREATE TABLE profiles (
    user_id         TEXT PRIMARY KEY NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    native_language TEXT NULL,
    goals           TEXT NULL,
    timezone        TEXT NULL,
    avatar_url      TEXT NULL
) STRICT;

-- +goose Down
DROP TABLE profiles;
DROP TABLE users;
