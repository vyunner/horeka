CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS products_name_trgm_idx
    ON public.products USING gin (lower(name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS product_aliases_alias_trgm_idx
    ON public.product_aliases USING gin (lower(alias) gin_trgm_ops);