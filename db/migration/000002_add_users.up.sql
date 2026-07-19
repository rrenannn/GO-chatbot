CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    whatsmeow_jid TEXT, -- JID do dispositivo pareado (preenchido após o primeiro login no WhatsApp)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE customers ADD COLUMN user_id UUID REFERENCES users(id);

-- Um mesmo número de telefone pode existir para clientes de contas diferentes
ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_phone_number_key;
ALTER TABLE customers ADD CONSTRAINT customers_user_phone_unique UNIQUE (user_id, phone_number);
