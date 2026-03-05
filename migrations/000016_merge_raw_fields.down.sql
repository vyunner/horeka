BEGIN;

ALTER TABLE public.order_requests
    ADD COLUMN raw_name TEXT,
    ADD COLUMN raw_amount TEXT;

UPDATE public.order_requests
SET raw_name   = SPLIT_PART(raw_product, ' ', 1),
    raw_amount = TRIM(SUBSTRING(raw_product FROM POSITION(' ' IN raw_product) + 1));

ALTER TABLE public.order_requests
    ALTER COLUMN raw_name SET NOT NULL,
ALTER COLUMN raw_amount SET NOT NULL;

ALTER TABLE public.order_requests
DROP COLUMN raw_product;

COMMIT;