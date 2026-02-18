CREATE TABLE roles (
                       id BIGSERIAL PRIMARY KEY,
                       name TEXT NOT NULL UNIQUE
);

INSERT INTO roles (name) VALUES
                             ('admin'),
                             ('manager')