CREATE TABLE customers (
                           id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                           phone_number VARCHAR(20) UNIQUE NOT NULL, -- Ex: 5511999999999
                           name VARCHAR(100) NOT NULL,
                           created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE chat_sessions (
                               id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                               customer_id UUID REFERENCES customers(id),
                               status session_status DEFAULT 'IDLE',
                               started_at TIMESTAMPTZ DEFAULT NOW(),
                               last_interaction_at TIMESTAMPTZ DEFAULT NOW(),
                               assigned_agent VARCHAR(100) NULL -- Nome/ID do atendente humano, se houver
);

CREATE TABLE message_history (
                                 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                 session_id UUID REFERENCES chat_sessions(id),
                                 sender_type VARCHAR(20) CHECK (sender_type IN ('BOT', 'USER', 'HUMAN')),
                                 content TEXT NOT NULL,
                                 created_at TIMESTAMPTZ DEFAULT NOW()
);