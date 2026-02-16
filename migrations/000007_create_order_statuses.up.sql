CREATE TABLE order_statuses (
                                id BIGSERIAL PRIMARY KEY,
                                name TEXT NOT NULL UNIQUE
);
