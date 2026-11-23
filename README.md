# mc-server-backend

Go backend for a Minecraft server management app. Provides authentication, user
management, server lifecycle control, mod management via Modrinth, and a
GitHub-based auto-updater.

## Requirements

- Go 1.24.4 (per `go.mod`)
- Write access to the SQLite file and Minecraft server data directory
- Java on the host (`java` on `PATH`, or `JAVA_8`/`JAVA_17`/`JAVA_21`/`JAVA_25`);
  MC ≤1.16.5 (incl. Forge 1.12.2–1.16.5) needs Java 8, 1.17–1.20.4 Java 17,
  1.20.5–1.21.x Java 21, 26.x Java 25
- Network access for Mojang, Fabric meta, Forge/NeoForge maven, Modrinth and
  GitHub releases (optional but
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

## Deployment

### 1. Install Java

The backend starts every Minecraft server with Java, so the host needs the
Java versions for the Minecraft versions you plan to offer:

| Minecraft version | Java |
|---|---|
| ≤ 1.16.5 (incl. Forge 1.12.2–1.16.5) | 8 |
| 1.17 – 1.20.4 | 17 |
| 1.20.5 – 1.21.x | 21 |
| 26.x | 25 |

Any JDK/JRE works; [Eclipse Temurin](https://adoptium.net/) is a good default.

- Linux (Debian/Ubuntu, after adding the Adoptium apt repo):
  ```bash
  sudo apt install temurin-8-jre temurin-17-jre temurin-21-jre
  ```
- Windows: run the Temurin MSI installer for each version.

Then point each version at its `java` binary in `.env` (`JAVA_8`, `JAVA_17`,
`JAVA_21`, `JAVA_25`). Unset entries fall back to `java` on `PATH`. Check
with `"<path>" -version`.

### 2. Open the Minecraft port range

Each Minecraft server is assigned a port from **30000–30050** (TCP). Open
this range on the host firewall (and any cloud security group / router port
forward) so players can connect:

- Linux (ufw): `sudo ufw allow 30000:30050/tcp`
- Linux (firewalld):
  ```bash
  sudo firewall-cmd --permanent --add-port=30000-30050/tcp
  sudo firewall-cmd --reload
  ```
- Windows (PowerShell as Administrator):
  ```powershell
  New-NetFirewallRule -DisplayName "Minecraft servers" -Direction Inbound -Protocol TCP -LocalPort 30000-30050 -Action Allow
  ```

The panel port (`PORT`) only needs to be reachable by the reverse proxy.

### 3. Serve the frontend behind a reverse proxy

The frontend (`web/`, build with `npm run build` → `web/dist`) calls the API
with relative URLs, and the backend does not serve `web/dist`. Put both behind
one origin with nginx/Caddy: serve `web/dist` as static files (SPA fallback to
`index.html`) and proxy these paths to the backend: `/api`, `/Authentication`,
`/user`, `/logout`, `/op`, `/LOL-AmongUs`.

Then set in `.env`:

```dotenv
TRUSTED_PROXIES=127.0.0.1        # IP of the reverse proxy
SESSION_COOKIE_SECURE=true       # when the site is served over HTTPS
FRONTEND_BASE_URL=https://panel.example.com
```

`TRUSTED_PROXIES` matters: logins are bound to the client IP, and without it
every user would appear to come from the proxy's address.

## .env Configuration

Create a `.env` file in the repo root. Values are strings; use `true`/`false`
for booleans.

Example:

```dotenv
FRONTEND_BASE_URL=http://localhost:5173
PORT=3000
# LOGIN_VERIFY_BASE_URL=http://localhost:3000
DEBUG=false
UA_FILTER=true
# TRUSTED_PROXIES=127.0.0.1
# SESSION_COOKIE_SECURE=false
# CORS_ALLOW_ORIGINS=

SESSION_SECRET=change-me
CRYPTO_SECRET=
HMAC_SECRET=

OIDC_GOOGLE_CLIENT_ID=__PLACEHOLDER__
OIDC_CUSTOM_CLIENT_ID=fgf-mc-panel
OIDC_CUSTOM_DISCOVERY_URL=http://localhost:15515/.well-known/openid-configuration
OIDC_REQUIRE_EMAIL_VERIFIED=true

# FGF login (all four required when using this flow)
# FGF_IDP_ISSUER=https://idp.example.com
# FGF_IDP_CLIENT_ID=__PLACEHOLDER__
# FGF_IDP_CLIENT_SECRET=__PLACEHOLDER__
# FGF_IDP_REDIRECT_URL=https://panel.example.com/login/fgf/callback

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

# Version lists and server installation (leave commented to use defaults)
# VERSION_CACHE_TTL=6h
# INSTALLER_TIMEOUT=10m
# INSTALLER_CONCURRENCY=2
# Java per Minecraft version (Windows paths such as C:\Program Files\Java\jdk-21\bin\java.exe)
# JAVA_8=
# JAVA_17=
# JAVA_21=
# JAVA_25=

# Modpack import defaults (leave commented to use defaults)
# MRPACK_MAX_SIZE_MB=150
# MRPACK_BLACKLIST=options.txt,optionsof.txt,servers.dat,servers.dat_old,resourcepacks/,shaderpacks/,screenshots/,config/*-client.*,config/iris.properties,config/sodium-options.json
# MRPACK_DOWNLOAD_HOSTS=cdn.modrinth.com,github.com,raw.githubusercontent.com,gitlab.com,release-assets.githubusercontent.com,objects.githubusercontent.com

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
- `LOGIN_VERIFY_BASE_URL`: Base URL used for login email verification links.
  Defaults to `http://localhost:${PORT}` (`PORT` defaults to `3000`).
- `DEBUG`: Enables debug mode and verbose logging when `true`.
- `UA_FILTER`: When `true`, applies User-Agent filtering (in debug mode it can
  be bypassed by setting this to `false`).
- `TRUSTED_PROXIES`: Comma-separated IPs/CIDRs of reverse proxies whose
  `X-Forwarded-For` is trusted for the client IP. Empty trusts no proxy (the
  TCP peer address is used), so forged `X-Forwarded-For` headers are ignored.
- `SESSION_COOKIE_SECURE`: When `true`, the session cookie is only sent over
  HTTPS. Leave `false` for plain-HTTP local development.
- `CORS_ALLOW_ORIGINS`: Extra comma-separated origins (e.g.
  `https://tools.example.com`) allowed to call the API cross-origin with
  credentials. `FRONTEND_BASE_URL` is always allowed. Not needed when the
  frontend is served from the same origin.
  With `DEBUG=true` every origin is allowed (local Vite dev server).

Secrets:
- `SESSION_SECRET`: Session cookie signing key. Must be a strong random string.
  If set to `random_string` the app will refuse to start.
- `CRYPTO_SECRET`: Encryption key for sensitive data. Defaults to
  `SESSION_SECRET` if empty.
- `HMAC_SECRET`: HMAC key. Defaults to `SESSION_SECRET` if empty.

OIDC:
- `OIDC_GOOGLE_CLIENT_ID`: Google OAuth client ID for ID token `aud` check.
- `OIDC_CUSTOM_CLIENT_ID`: FGF-idp OAuth client ID for ID token `aud` check.
- `OIDC_CUSTOM_DISCOVERY_URL`: FGF-idp discovery URL
  (e.g. `http://localhost:15515/.well-known/openid-configuration`).
- `OIDC_REQUIRE_EMAIL_VERIFIED`: Require `email_verified=true` (default `true`).

FGF login (defaults are empty strings; configure all four for this flow):
- `FGF_IDP_ISSUER`: IdP issuer URL; trailing `/` is removed.
- `FGF_IDP_CLIENT_ID`: OAuth client ID string.
- `FGF_IDP_CLIENT_SECRET`: Client secret string, at least 32 characters.
- `FGF_IDP_REDIRECT_URL`: Callback URL registered with the IdP for the frontend
  FGF callback page. Issuer and redirect URLs require HTTPS, except on loopback.

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
- `VERSION_CACHE_TTL`: How long version lists (Mojang manifest, Fabric loaders,
  Forge/NeoForge maven metadata) are cached in memory. Go duration (`6h`, `30m`)
  or seconds; default `6h`. When a refresh fails the last good list is served.
- `INSTALLER_TIMEOUT`: Time limit for one Forge/NeoForge `--installServer` run.
  Go duration or seconds; default `10m`.
- `INSTALLER_CONCURRENCY`: Maximum Forge/NeoForge installers running at once
  (default `2`).
- `JAVA_8`, `JAVA_17`, `JAVA_21`, `JAVA_25`: Java executables used to launch
  servers and run installers. The required Java comes from Mojang's version data
  (`javaVersion.majorVersion`; e.g. 1.16.5 → 8, 1.17 → 16 → `JAVA_17`,
  1.20.5–1.21.x → 21, 26.x → 25). Unset (default) uses `java` from `PATH`. When none are set no
  version data is fetched at start.

Modpack import:
- `MRPACK_MAX_SIZE_MB`: Integer archive size limit in MiB (1 MiB = 1,048,576
  bytes); default `150`. Unset, empty or invalid integers use the default.
- `MRPACK_BLACKLIST`: Comma-separated instance-relative paths, directory
  prefixes ending in `/`, or Go `path.Match` patterns; surrounding whitespace
  is ignored. When **unset**, the default is
  `options.txt,optionsof.txt,servers.dat,servers.dat_old,resourcepacks/,shaderpacks/,screenshots/,config/*-client.*,config/iris.properties,config/sodium-options.json`.
  Setting **an empty string** (`MRPACK_BLACKLIST=`) **disables the blacklist**.
  Files marked `env.server=unsupported` are always excluded independently.
- `MRPACK_DOWNLOAD_HOSTS`: Comma-separated exact hostnames (no scheme, port or
  path), with surrounding whitespace ignored. Default:
  `cdn.modrinth.com,github.com,raw.githubusercontent.com,gitlab.com,release-assets.githubusercontent.com,objects.githubusercontent.com`.
  Unset or empty uses the default; a nonempty value replaces the entire list.
  Downloads require HTTPS.

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

## Modpack import

Server-side `.mrpack` import with local preview, file selection, install sessions and progress: [usage and API](docs/mrpack-import.md).
