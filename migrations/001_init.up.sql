CREATE TYPE message_status AS ENUM ('sent', 'delivered', 'read');

CREATE TABLE IF NOT EXISTS chats (
    id UUID PRIMARY KEY,
    user1_id UUID NOT NULL,
    user2_id UUID NOT NULL,
    last_message_id BIGINT NOT NULL DEFAULT 0,
    last_message TEXT,
    last_message_at TIMESTAMP DEFAULT NOW(),
    last_message_status message_status,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (user1_id, user2_id),
    CHECK (user1_id < user2_id)
);

CREATE TABLE IF NOT EXISTS messages (
    id BIGINT NOT NULL,
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL,
    content TEXT NOT NULL,
    status message_status NOT NULL DEFAULT 'sent',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (chat_id, id)
);

CREATE TABLE IF NOT EXISTS chat_user_state (
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    last_read_message_id BIGINT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_chats_user1_last_message_at
    ON chats (user1_id, last_message_at DESC);

CREATE INDEX IF NOT EXISTS idx_chats_user2_last_message_at
    ON chats (user2_id, last_message_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_chat_id_desc
    ON messages (chat_id, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_user_state_user
    ON chat_user_state (user_id);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_messages_set_updated_at ON messages;
CREATE TRIGGER trg_messages_set_updated_at
BEFORE UPDATE ON messages
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE FUNCTION validate_message_status_transition()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status < OLD.status THEN
        RAISE EXCEPTION 'invalid status transition: % -> %', OLD.status, NEW.status;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_messages_validate_status_transition ON messages;
CREATE TRIGGER trg_messages_validate_status_transition
BEFORE UPDATE OF status ON messages
FOR EACH ROW
EXECUTE FUNCTION validate_message_status_transition();

CREATE OR REPLACE FUNCTION sync_chat_preview()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE chats
    SET
        last_message_id = NEW.id,
        last_message = NEW.content,
        last_message_at = NEW.created_at,
        last_message_status = NEW.status
    WHERE id = NEW.chat_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_messages_sync_chat_preview ON messages;
CREATE TRIGGER trg_messages_sync_chat_preview
AFTER INSERT ON messages
FOR EACH ROW
EXECUTE FUNCTION sync_chat_preview();

