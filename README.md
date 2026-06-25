# Go SIEM Agent — LLM Classifier

A Security Information and Event Management (SIEM) agent built in Go that uses LLM-based classification to analyze and triage security log events. Includes a React dashboard for real-time monitoring.

## Features

- **LLM-powered classification** — uses an OpenAI-compatible LLM (default: Kimi K2) to classify security events by severity and MITRE ATT&CK technique
- **Multi-format log parsing** — supports syslog and JSON log formats
- **Vector similarity search** — stores event embeddings in Qdrant for finding similar past events
- **Prometheus metrics** — built-in `/metrics` endpoint for observability
- **REST API** — Chi-based HTTP server with Swagger docs
- **React dashboard** — drag-and-drop log ingestion, event cards, analytics panel, and IOC list

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, Chi router |
| LLM | Kimi K2 (OpenAI-compatible) |
| Vector DB | Qdrant |
| Embeddings | Ollama |
| Metrics | Prometheus |
| Frontend | React, TypeScript, Vite, Tailwind CSS |

## Project Structure

```
siemagent/
├── cmd/siemagent/       # Main entrypoint
├── internal/
│   ├── api/             # HTTP handlers, server, Swagger
│   ├── classifier/      # LLM-based event classifier
│   ├── config/          # Environment config
│   ├── metrics/         # Prometheus metrics
│   ├── models/          # Shared data models
│   ├── parser/          # Syslog & JSON log parsers
│   ├── pipeline/        # Worker pool
│   └── store/           # In-memory event store
├── pkg/
│   ├── ollama/          # Ollama embeddings client
│   └── qdrant/          # Qdrant vector DB client
└── web/                 # React frontend
    └── src/
        ├── components/  # EventCard, IOCList, DropZone, etc.
        └── pages/       # Dashboard, Search
```

## Getting Started

### Prerequisites

- Go 1.25+
- Node.js 18+
- [Qdrant](https://qdrant.tech/) running locally
- [Ollama](https://ollama.com/) running locally (for embeddings)
- Kimi K2 API key (or any OpenAI-compatible provider)

### Setup

1. Clone the repo:
   ```sh
   git clone https://github.com/ChanchalS7/go-siem-agent-llm-classifier.git
   cd go-siem-agent-llm-classifier/siemagent
   ```

2. Copy and configure environment variables:
   ```sh
   cp .env.example .env
   # Edit .env with your API keys and endpoints
   ```

3. Start dependencies with Docker:
   ```sh
   docker-compose up -d
   ```

4. Pull Ollama embedding models:
   ```sh
   ./scripts/pull-models.sh
   ```

5. Run the backend:
   ```sh
   make run
   ```

6. Run the frontend:
   ```sh
   cd web
   npm install
   npm run dev
   ```

The API will be available at `http://localhost:8080` and the dashboard at `http://localhost:5173`.

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `KIMCHI_API_KEY` | LLM provider API key | — |
| `KIMCHI_BASE_URL` | LLM provider base URL | `https://api.kimchi.ai/v1` |
| `SIEM_MODEL` | Model ID to use | `kimi-k2-5` |
| `CONDUCTOR_PORT` | HTTP server port | `8080` |
| `ALLOWED_ORIGIN` | CORS allowed origin | `http://localhost:5173` |
| `QDRANT_ADDR` | Qdrant gRPC address | `localhost:6334` |
| `OLLAMA_BASE_URL` | Ollama base URL | `http://localhost:11434` |

## API

Swagger UI is available at `http://localhost:8080/swagger` when the server is running.

## Running Tests

```sh
cd siemagent
make test
```

## License

MIT
