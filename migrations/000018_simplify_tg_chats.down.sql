ALTER TABLE tg_chats DROP CONSTRAINT tg_chats_chat_id_key;
ALTER TABLE tg_chats ADD COLUMN location_id BIGINT REFERENCES locations(id) ON DELETE CASCADE;
ALTER TABLE tg_chats ADD CONSTRAINT tg_chats_chat_id_location_id_key UNIQUE (chat_id, location_id);
