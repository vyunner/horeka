CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     phone TEXT NOT NULL UNIQUE,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );
