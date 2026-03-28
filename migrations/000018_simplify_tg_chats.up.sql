-- Remove duplicate chat_ids before dropping location_id
DELETE FROM tg_chats a USING tg_chats b
WHERE a.id > b.id AND a.chat_id = b.chat_id;

ALTER TABLE tg_chats DROP CONSTRAINT tg_chats_chat_id_location_id_key;
ALTER TABLE tg_chats DROP COLUMN location_id;
ALTER TABLE tg_chats ADD CONSTRAINT tg_chats_chat_id_key UNIQUE (chat_id);
