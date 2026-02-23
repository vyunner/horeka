-- Добавляем себестоимость строк заказа
alter table public.order_products
    add column price_cost integer,
    add column total_sum_cost integer;

-- Добавляем себестоимость заказа
alter table public.orders
    add column total_sum_cost integer default 0 not null;

-- Ограничения от мусорных значений
alter table public.order_products
    add constraint order_products_price_cost_check
        check (price_cost is null or price_cost >= 0),
    add constraint order_products_total_sum_cost_check
        check (total_sum_cost is null or total_sum_cost >= 0);

alter table public.orders
    add constraint orders_total_sum_cost_check
        check (total_sum_cost >= 0);