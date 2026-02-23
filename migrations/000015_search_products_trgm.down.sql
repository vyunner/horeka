DROP INDEX IF EXISTS public.product_aliases_alias_trgm_idx;
DROP INDEX IF EXISTS public.products_name_trgm_idx;

-- extension обычно не удаляют, потому что она может использоваться другими индексами
-- если всё же нужно:
-- DROP EXTENSION IF EXISTS pg_trgm;