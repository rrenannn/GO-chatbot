# 🤖 Gocase WhatsApp AI Bot

Este é um assistente virtual de atendimento automatizado para a Gocase, desenvolvido em Go.  
O bot utiliza a inteligência artificial do Gemini para processar solicitações de clientes, resolver problemas comuns de pós-venda e realizar o transbordo para atendimento humano quando necessário.

---

## 🚀 Funcionalidades

- **Atendimento Automatizado (Gabi):** Persona amigável e empática baseada no manual oficial  
- **Gestão de Problemas Críticos:** Fluxos inteligentes para atrasos, modelos errados, produtos riscados, problemas de senha, entre outros  
- **Auto-Cadastro:** Identifica novos números e cadastra o cliente automaticamente no banco de dados  
- **Máquina de Estados:** Mantém o contexto da conversa, lembrando do problema relatado após o envio dos dados do pedido  
- **Transbordo Humano:** Detecta frustração ou solicitações complexas e encaminha para especialistas  
- **Arquitetura Limpa:** Separação clara entre entrega (WhatsApp/HTTP), casos de uso e repositório  

---

## 🛠️ Tecnologias Utilizadas

### Linguagem
- Go (Golang)

### Inteligência Artificial
- Google Gemini 2.5 Flash (via Generative AI SDK)

### WhatsApp
- Whatsmeow (Protocolo WebSocket)

### Banco de Dados
- PostgreSQL → Armazenamento de clientes, sessões e histórico  
- SQLite (Pure Go) → Gestão de sessões do WhatsApp (`modernc.org/sqlite`, sem CGO)

### Ferramentas
- SQLC → Geração de código Go a partir de SQL  
- Golang-Migrate → Controle de versão do banco  
- Echo → Framework web para rotas HTTP  

---

## 📋 Pré-requisitos

- Go instalado  
- PostgreSQL rodando (Docker recomendado)  
- Chave de API do Google Gemini  

---

## 🔧 Configuração

### 1. Clone o repositório

```bash
git clone https://github.com/seu-usuario/GO-chatbot.git
cd GO-chatbot
```

### 2. Configure as variáveis de ambiente

Crie um arquivo `.env` na raiz:

```env
DATABASE_URL=postgres://user:pass@localhost:5432/gocase_bot?sslmode=disable
GEMINI_API_KEY=sua_chave_aqui
PORT=8080
```

### 3. Execute as migrações

```bash
migrate -path sql/schema -database "postgres://user:pass@localhost:5432/gocase_bot?sslmode=disable" up
```

### 4. Gere o código com SQLC

```bash
sqlc generate
```

---

## 🏃 Como Rodar

1. Execute a aplicação:

```bash
go run cmd/api/main.go
```

2. Abra o arquivo de teste (`index.html`) no navegador  
3. Escaneie o QR Code com seu WhatsApp  
4. O bot estará pronto para uso 🚀  

---

## 🧠 Fluxos de Atendimento

O bot está preparado para lidar com os principais cenários de pós-venda:

1. **Pedido Atrasado** → Nova previsão + priorização  
2. **Entrega não Recebida** → Abertura de acareação  
3. **Endereço Errado** → Validação de alteração  
4. **Rastreio Inválido** → Explicação de atualização logística  
5. **Erro de Personalização/Modelo** → Solicitação de fotos + troca  
6. **Produto com Avaria** → Tratamento prioritário  
7. **Prazo de Troca** → Política + exceções  
8. **Senha/Acesso** → Recuperação de conta  

---

## 📄 Licença

Este projeto é destinado a fins de estudo e demonstração técnica de integração entre IA e mensageria.

---

## 💡 Dica

Garanta que seu `.gitignore` contenha:

```
/vendor/
.env
sessions.db*
api_test.html
```
