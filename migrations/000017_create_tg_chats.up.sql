CREATE TABLE IF NOT EXISTS tg_chats (
    id          BIGSERIAL PRIMARY KEY,
    chat_id     BIGINT NOT NULL,
    location_id BIGINT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(chat_id, location_id)
);
