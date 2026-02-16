CREATE TABLE orders (
                        id BIGSERIAL PRIMARY KEY,
                        user_id BIGINT NOT NULL REFERENCES users(id),
                        location_id BIGINT NOT NULL REFERENCES locations(id),
                        status_id BIGINT NOT NULL REFERENCES order_statuses(id),
                        total_sum NUMERIC(12,2) NOT NULL DEFAULT 0,
                        comment TEXT,
                        created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_location_created
    ON orders(location_id, created_at);

CREATE INDEX idx_orders_user_created
    ON orders(user_id, created_at);

CREATE INDEX idx_orders_status
    ON orders(status_id);
