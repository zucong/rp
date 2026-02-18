# Roleplay Chat 🤖

A local LLM-powered roleplay group chat application. Create custom characters, manage multiple rooms, and enjoy immersive AI-driven conversations.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react)](https://reactjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0-3178C6?style=flat-square&logo=typescript)](https://www.typescriptlang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)

## ✨ Features

- 🎭 **Custom Characters** - Create characters with names, avatars, system prompts, and model parameters
- 🏠 **Multi-Room Management** - Each room has its own context and participants
- 🤖 **Smart Selection** - Orchestrator automatically decides which characters participate in responses
- 💬 **Force Control** - Use `@CharacterName` to force include, `!CharacterName` to force exclude
- 👤 **User Participation** - Play as characters and join the group chat
- 🔧 **Flexible Configuration** - Supports OpenAI, OpenRouter, Azure, Ollama, and other OpenAI-compatible APIs
- 📊 **Debug Tools** - LLM call logs and orchestrator decision tree visualization

## 🚀 Quick Start

### Requirements

- Go 1.23+ (tested with 1.25.6)
- Node.js 18+ (tested with 24.x)
- npm or yarn

### Development Mode

```bash
# Clone the repository
git clone https://github.com/zucong/rp.git
cd rp

# Configuration
# 1. Copy the example config file
cd backend
cp config.example.yaml config.yaml
# 2. Edit config.yaml and add your API key

# Start the backend server
go run .

# New terminal - start the frontend
cd ../frontend
npm install
npm run dev
```

Visit http://localhost:5173

### Alternative: Using Make

```bash
# Start both backend and frontend in development mode
make dev

# Or start them separately
make dev-backend   # Terminal 1
make dev-frontend  # Terminal 2

# Check service status
make status

# Stop all services
make stop
```

### Production Build

```bash
# Build frontend and backend
make build

# Run the backend (serves static frontend files from frontend/dist/)
./bin/rp
```

Then visit http://localhost:8080

## 📖 Usage Guide

### 1. Configure LLM API

Configure your LLM API on first use:

| Setting | Example Value |
|---------|---------------|
| API Endpoint | `https://api.openai.com/v1` or `http://localhost:11434/v1` (Ollama) |
| API Key | Your API key |
| Default Model | `gpt-3.5-turbo` / `gpt-4` / `llama2` etc. |

### 2. Create Characters

- Set character name and system prompt
- Configure model parameters (temperature, max tokens, etc.)
- Mark whether users can play this character

### 3. Create Rooms

- Create a new room and set the background description
- Add AI characters and user-playable characters

### 4. Start Chatting

- Enter a room and start the conversation
- Use `@CharacterName` to force a character to respond
- Use `!CharacterName` to force exclude a character
- Click the message actions to view LLM logs or decision process

## 🏗️ Project Structure

```
.
├── backend/          # Go backend
│   ├── handlers/     # HTTP handlers
│   ├── models/       # Data models
│   ├── db/           # Database
│   ├── llm/          # LLM client
│   └── services/     # Business logic (LLM logging, decision recording)
├── frontend/         # React frontend
│   ├── src/pages/    # Page components
│   └── src/components/  # UI components (custom, Tailwind-based)
├── data/             # SQLite database (auto-created)
└── bin/              # Build output
```

## 🛠️ Tech Stack

- **Backend**: Go + Gin + sqlx + SQLite
- **Frontend**: React + TypeScript + Vite + Tailwind CSS + Radix UI primitives
- **Real-time Communication**: Server-Sent Events (SSE)
- **LLM**: OpenAI-compatible API

## 📝 API Documentation

### Characters
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/characters` | GET | List all characters |
| `/api/characters` | POST | Create a character |
| `/api/characters/:id` | GET | Get a character |
| `/api/characters/:id` | PUT | Update a character |
| `/api/characters/:id` | DELETE | Delete a character |

### Rooms
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/rooms` | GET | List all rooms |
| `/api/rooms` | POST | Create a room |
| `/api/rooms/:id` | GET | Get a room |
| `/api/rooms/:id` | PUT | Update a room |
| `/api/rooms/:id` | DELETE | Delete a room |
| `/api/rooms/:id/participants` | GET | List room participants |
| `/api/rooms/:id/participants` | POST | Add a participant |
| `/api/rooms/:id/participants/:pid` | DELETE | Remove a participant |
| `/api/rooms/:id/messages` | GET | Get room messages |
| `/api/rooms/:id/messages` | DELETE | Clear all messages |

### Chat
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/rooms/:id/chat` | POST | Send a message |
| `/api/rooms/:id/events` | GET | SSE stream for real-time updates |
| `/api/rooms/:id/regenerate` | POST | Regenerate AI responses |
| `/api/messages/:msgId` | PUT | Edit a message |
| `/api/messages/:msgId` | DELETE | Delete a message |

### Debug
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/messages/:msgId/llm-logs` | GET | Get LLM call logs |
| `/api/messages/:msgId/decisions` | GET | Get orchestrator decision tree |

### Config
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/config` | GET | Get system configuration |
| `/api/config` | PUT | Update system configuration |

## 🤝 Contributing

Issues and Pull Requests are welcome!

## 📄 License

[MIT](LICENSE) © zucong
