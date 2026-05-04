-- Enum para controlar o estado da conversa
CREATE TYPE session_status AS ENUM (
    'IDLE',                 -- Sem conversa ativa
    'WAITING_USER_REPLY',   -- Bot enviou a mensagem de pós-venda e aguarda
    'AI_HANDLING',          -- IA está conversando com o cliente
    'HUMAN_HANDLING',       -- Transbordo ocorreu, bot pausado
    'RESOLVED'              -- Atendimento finalizado
);

CREATE TABLE IF NOT EXISTS customers (
                           id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                           phone_number VARCHAR(20) UNIQUE NOT NULL, -- Ex: 5511999999999
                           name VARCHAR(100) NOT NULL,
                           created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chat_sessions (
                               id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                               customer_id UUID REFERENCES customers(id),
                               status session_status DEFAULT 'IDLE',
                               started_at TIMESTAMPTZ DEFAULT NOW(),
                               last_interaction_at TIMESTAMPTZ DEFAULT NOW(),
                               assigned_agent VARCHAR(100) NULL -- Nome/ID do atendente humano, se houver
);

CREATE TABLE IF NOT EXISTS message_history (
                                 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                 session_id UUID REFERENCES chat_sessions(id),
                                 sender_type VARCHAR(20) CHECK (sender_type IN ('BOT', 'USER', 'HUMAN')),
                                 content TEXT NOT NULL,
                                 created_at TIMESTAMPTZ DEFAULT NOW()
);