# WASAText

A real-time messaging web application built for the [Web and Software Architecture](http://gamificationlab.uniroma1.it/en/wasa/) course. WASAText supports user authentication, 1-on-1 and group conversations, text & photo messages, emoji reactions, and live updates via WebSockets.

## Tech Stack

| Layer    | Technology                                               |
| -------- | -------------------------------------------------------- |
| Backend  | Go 1.25 · [httprouter](https://github.com/julienschmidt/httprouter) · Gorilla WebSocket |
| Database | SQLite (via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)) |
| Frontend | Vue 3 · Vue Router · Axios · Vite                        |
| Infra    | Docker Compose · Nginx (frontend) · Alpine images        |

## Project Structure

```
wasa-project/
├── cmd/
│   ├── healthcheck/          # Health-check daemon
│   └── webapi/               # API server entry point & configuration
│       ├── main.go            
│       ├── cors.go            
│       ├── load-configuration.go
│       └── register-web-ui*.go
├── service/
│   ├── api/                  # REST + WebSocket handlers, auth middleware
│   │   ├── handlers.go        # All HTTP endpoint handlers
│   │   ├── auth.go            # Bearer-token authentication
│   │   ├── websocket.go       # Real-time WebSocket hub
│   │   └── ...
│   ├── database/             # SQLite data-access layer
│   │   ├── database.go        # Schema init & connection
│   │   ├── user.go            # User queries
│   │   ├── conversation.go    # Conversation queries
│   │   ├── message.go         # Message queries
│   │   └── reaction.go        # Reaction queries
│   └── globaltime/           # Testable time wrapper
├── webui/                    # Vue 3 single-page application
│   ├── src/
│   │   ├── views/
│   │   │   ├── LoginView.vue
│   │   │   ├── HomeView.vue
│   │   │   ├── ConversationsView.vue
│   │   │   ├── ChatView.vue
│   │   │   └── ProfileView.vue
│   │   ├── components/        # ErrorMsg, GroupInfoPanel, LoadingSpinner
│   │   ├── services/          # api.js (API client), axios.js
│   │   ├── router/            # Vue Router config
│   │   └── assets/
│   ├── public/
│   ├── nginx.conf             # Production Nginx config
│   ├── package.json
│   └── vite.config.js
├── doc/
│   └── api.yaml              # OpenAPI 3 specification
├── demo/
│   └── config.yml            # Example configuration
├── data/                     # Runtime SQLite database (git-ignored)
├── uploads/                  # User-uploaded files
├── Dockerfile.backend        # Multi-stage Go build → Alpine
├── Dockerfile.frontend       # Multi-stage Node build → Nginx
├── docker-compose.yml        # Full-stack orchestration
├── go.mod / go.sum
└── vendor/                   # Go vendored dependencies
```

## Getting Started

### Prerequisites

* **Go** ≥ 1.25 (with CGO enabled for SQLite)
* **Node.js** ≥ 18 & **Yarn** ≥ 4
* **Docker** & **Docker Compose** (for containerised deployment)

### Run with Docker Compose (recommended)

```bash
docker compose up --build
```

| Service  | URL                    |
| -------- | ---------------------- |
| Frontend | http://localhost       |
| Backend  | http://localhost:3000  |

### Run Locally (development)

**Backend**

```bash
go run ./cmd/webapi/
```

The API server starts on port `3000` by default.

**Frontend**

```bash
cd webui
yarn install --immutable
yarn dev
```

Vite dev-server starts on http://localhost:5173 with hot-reload.

## Docker Build Process

The project uses **multi-stage Docker builds** to produce small, production-ready images.

### Backend (`Dockerfile.backend`)

| Stage     | Base Image              | What happens                                              |
| --------- | ----------------------- | --------------------------------------------------------- |
| **build** | `golang:1.25.1-alpine`  | Installs GCC & SQLite libs, compiles the Go binary with CGO |
| **run**   | `alpine:latest`         | Copies only the binary + SQLite runtime libs (~30 MB)     |

Build individually:

```bash
docker build -f Dockerfile.backend -t wasatext-backend .
```

The binary runs as:
```
./webapi --db-filename /data/wasatext.db --web-apihost 0.0.0.0:3000
```

### Frontend (`Dockerfile.frontend`)

| Stage     | Base Image          | What happens                                        |
| --------- | ------------------- | --------------------------------------------------- |
| **build** | `node:18-alpine`    | Installs deps via Yarn, runs `yarn run build-prod`  |
| **run**   | `nginx:alpine`      | Serves the `dist/` bundle with a custom `nginx.conf`|

Build individually:

```bash
docker build -f Dockerfile.frontend -t wasatext-frontend .
```

### Docker Compose Architecture

`docker-compose.yml` wires everything together:

```
┌──────────────┐        ┌──────────────┐
│   frontend   │──────▶│   backend    │
│  (nginx:80)  │  proxy │  (go:3000)   │
└──────────────┘        └──────┬───────┘
                               │
                    ┌──────────┴───────────┐
                    │  wasatext-data vol    │  ← SQLite DB
                    │  ./uploads bind mount │  ← uploaded files
                    └──────────────────────┘
```

Key details:

* **Network** — both containers share a `wasatext-network` bridge so the frontend can reverse-proxy API calls to `backend:3000`.
* **Volumes** — a named volume `wasatext-data` persists the SQLite database at `/data`; the host `./uploads` directory is bind-mounted to `/app/uploads`.
* **Health check** — the backend container has a built-in health check that hits `GET /liveness` every 30 s.

### Useful Commands

```bash
# Build & start everything
docker compose up --build

# Rebuild only the backend
docker compose up --build backend

# Stop & remove containers (keeps volumes)
docker compose down

# Stop & remove containers AND volumes (fresh start)
docker compose down -v

# View live logs
docker compose logs -f
```

## API Documentation

The full OpenAPI 3 specification lives in [`doc/api.yaml`](doc/api.yaml). You can preview it by opening `doc/index.html` in a browser or pasting the YAML into [Swagger Editor](https://editor.swagger.io/).

## Go Vendoring

This project uses [Go Vendoring](https://go.dev/ref/mod#vendoring). After changing dependencies (`go get` or `go mod tidy`), run:

```bash
go mod vendor
```

and commit the updated `vendor/` directory.

## Building for Production

**Backend only**

```bash
go build ./cmd/webapi/
```

**Frontend only**

```bash
cd webui
yarn run build-prod
```

The production bundle is output to `webui/dist/`.

## License

See [LICENSE](LICENSE).
