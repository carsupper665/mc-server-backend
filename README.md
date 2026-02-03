# mc-server-backend

Go backend for a Minecraft server management app. Provides authentication, user
management, server lifecycle control, mod management via Modrinth, and a
GitHub-based auto-updater.

## Requirements

- Go 1.24.4 (per `go.mod`)
- Write access to the SQLite file and Minecraft server data directory
- Network access for Modrinth/Fabric meta and GitHub releases (optional but
  required for mod install/update and auto-update)

## Install and Run

1. Create a `.env` file in the project root (see below).
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Start the server:
   ```bash
   go run main.go
   ```
4. The HTTP server listens on `:${PORT}` (default `3000`).

Optional build/test:

```bash
go build ./...
go test ./...
```

## .env Configuration

Create a `.env` file in the repo root. Values are strings; use `true`/`false`
for booleans.

Example:

```dotenv
FRONTEND_BASE_URL=http://localhost:5173
PORT=3000
DEBUG=false
UA_FILTER=true

SESSION_SECRET=change-me
CRYPTO_SECRET=
HMAC_SECRET=

SQLITE_PATH=DB.db?_busy_timeout=5000
SQL_MAX_IDLE_CONNS=100
SQL_MAX_OPEN_CONNS=1000
SQL_MAX_LIFETIME=60

MEMORY_CACHE_ENABLED=false
SYNC_FREQUENCY=60
BATCH_UPDATE_INTERVAL=5
RELAY_TIMEOUT=0

GLOBAL_API_RATE_LIMIT=60
GLOBAL_API_RATE_LIMIT_DURATION=60
DC_WEBHOOK_URL=

MINECRAFT_SERVER_PATH=./minecraft_servers
LATEST_FABRIC_LOADER_VERSION=
LATEST_FABRIC_INSTALLER_VERSION=1.1.0

AUTO_UPDATE=false
UPDATE_BETA=false

CREATE_ROOT_USER=false
ROOT_USER_EMAIL=
ROOT_USER_PASSWORD=
ROOT_USER_NAME=root

SMTP_SERVER=
SMTP_PORT=587
SMTP_SSL_ENABLED=false
SMTP_ACCOUNT=
SMTP_FROM=
SMTP_TOKEN=

NUM=5
CHANCE=1000
```

### Option Details

Core:
- `FRONTEND_BASE_URL`: Base URL for frontend. Used for NoRoute redirect.
- `PORT`: HTTP listen port. If empty, uses `-port` flag (default `3000`).
- `DEBUG`: Enables debug mode and verbose logging when `true`.
- `UA_FILTER`: When `true`, applies User-Agent filtering (in debug mode it can
  be bypassed by setting this to `false`).

Secrets:
- `SESSION_SECRET`: Session cookie signing key. Must be a strong random string.
  If set to `random_string` the app will refuse to start.
- `CRYPTO_SECRET`: Encryption key for sensitive data. Defaults to
  `SESSION_SECRET` if empty.
- `HMAC_SECRET`: HMAC key. Defaults to `SESSION_SECRET` if empty.

Database:
- `SQLITE_PATH`: SQLite DSN/path (default `DB.db?_busy_timeout=5000`).
- `SQL_MAX_IDLE_CONNS`: Max idle DB connections (default `100`).
- `SQL_MAX_OPEN_CONNS`: Max open DB connections (default `1000`).
- `SQL_MAX_LIFETIME`: Max connection lifetime in seconds (default `60`).

Caching and background tasks:
- `MEMORY_CACHE_ENABLED`: Enables in-memory cache when `true`.
- `SYNC_FREQUENCY`: Background sync interval in seconds (default `60`).
- `BATCH_UPDATE_INTERVAL`: Batch update interval in seconds (default `5`).
- `RELAY_TIMEOUT`: Relay timeout in seconds (default `0`).

Rate limiting and alerts:
- `GLOBAL_API_RATE_LIMIT`: Max requests per IP per window (default `60`).
- `GLOBAL_API_RATE_LIMIT_DURATION`: Window size in seconds (default `60`).
- `DC_WEBHOOK_URL`: Discord webhook URL for panic/error notifications.

Minecraft server settings:
- `MINECRAFT_SERVER_PATH`: Base directory for server files
  (default `./minecraft_servers`).
- `LATEST_FABRIC_LOADER_VERSION`: Override Fabric loader version (optional).
- `LATEST_FABRIC_INSTALLER_VERSION`: Override Fabric installer version
  (default `1.1.0` when override is needed).

Auto update:
- `AUTO_UPDATE`: Enable GitHub release auto-update when `true`.
- `UPDATE_BETA`: If `true`, use beta/prerelease tags when auto-updating.

Root user bootstrap:
- `CREATE_ROOT_USER`: When `true`, auto-creates a root user on startup.
- `ROOT_USER_EMAIL`: Required if `CREATE_ROOT_USER=true`.
- `ROOT_USER_PASSWORD`: Optional; default `123456`.
- `ROOT_USER_NAME`: Optional; default `root`.

SMTP (email):
- `SMTP_SERVER`: SMTP host.
- `SMTP_PORT`: SMTP port (default `587`).
- `SMTP_SSL_ENABLED`: Enable SSL/TLS when `true`.
- `SMTP_ACCOUNT`: SMTP username.
- `SMTP_FROM`: From address.
- `SMTP_TOKEN`: SMTP password or app token.

Among Us mini-game:
- `NUM`: Player limit (default `5`).
- `CHANCE`: Impostor chance threshold (default `1000`).

## Project Notes

- Logs default to `./logs` and can be changed with `-log-dir`.
