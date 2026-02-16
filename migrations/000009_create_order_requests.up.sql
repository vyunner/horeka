CREATE TABLE order_requests (
                                id BIGSERIAL PRIMARY KEY,
                                order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
                                raw_name TEXT NOT NULL,
                                raw_amount TEXT NOT NULL,
                                is_available BOOLEAN,
                                created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_requests_order
    ON order_requests(order_id);
