CREATE TABLE order_products (
                                id BIGSERIAL PRIMARY KEY,
                                order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
                                request_id BIGINT REFERENCES order_requests(id) ON DELETE SET NULL,
                                product_id BIGINT NOT NULL REFERENCES products(id),
                                quantity NUMERIC(10,3) NOT NULL,
                                unit TEXT NOT NULL,
                                price NUMERIC(10,2) NOT NULL,
                                total_sum NUMERIC(12,2) NOT NULL,
                                available BOOLEAN NOT NULL DEFAULT true,
                                created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_products_order
    ON order_products(order_id);

CREATE INDEX idx_order_products_product
    ON order_products(product_id);

CREATE INDEX idx_order_products_request
    ON order_products(request_id);
