BEGIN;

ALTER TABLE public.order_requests
    ADD COLUMN raw_product TEXT;

UPDATE public.order_requests
SET raw_product = raw_name || ' ' || raw_amount;

ALTER TABLE public.order_requests
    ALTER COLUMN raw_product SET NOT NULL;

ALTER TABLE public.order_requests
DROP COLUMN raw_name,
    DROP COLUMN raw_amount;

COMMIT;