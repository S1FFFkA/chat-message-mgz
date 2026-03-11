DROP TRIGGER IF EXISTS trg_messages_sync_chat_preview ON messages;
DROP TRIGGER IF EXISTS trg_messages_validate_status_transition ON messages;
DROP TRIGGER IF EXISTS trg_messages_set_updated_at ON messages;

DROP FUNCTION IF EXISTS sync_chat_preview();
DROP FUNCTION IF EXISTS validate_message_status_transition();
DROP FUNCTION IF EXISTS set_updated_at();

DROP INDEX IF EXISTS idx_chat_user_state_user;
DROP INDEX IF EXISTS idx_messages_chat_id_desc;
DROP INDEX IF EXISTS idx_chats_user2_last_message_at;
DROP INDEX IF EXISTS idx_chats_user1_last_message_at;

DROP TABLE IF EXISTS chat_user_state;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chats;

DROP TYPE IF EXISTS message_status;

