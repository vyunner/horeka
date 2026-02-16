CREATE TABLE product_aliases (
                                 id BIGSERIAL PRIMARY KEY,
                                 product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                                 alias TEXT NOT NULL
);

CREATE INDEX idx_product_aliases_product_id
    ON product_aliases(product_id);

CREATE INDEX idx_product_aliases_alias
    ON product_aliases(alias);
