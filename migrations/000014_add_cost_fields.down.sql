-- Удаляем ограничения
alter table public.orders
drop constraint if exists orders_total_sum_cost_check;

alter table public.order_products
drop constraint if exists order_products_total_sum_cost_check,
    drop constraint if exists order_products_price_cost_check;

-- Удаляем колонки
alter table public.orders
drop column if exists total_sum_cost;

alter table public.order_products
drop column if exists total_sum_cost,
    drop column if exists price_cost;