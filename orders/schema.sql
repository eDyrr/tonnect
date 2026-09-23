create table if not exists orders (
    id text primary key,
    status text not null check (status in ('created', 'paid', 'expired')),
    reference text not null,
    recipient text not null,
    amount bigint not null,
    paid_tx_hash text,
    created_at timestamptz not null default now(),
    paid_at timestamptz
);
alter table orders add column if not exists expires_at timestamptz;