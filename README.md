# Server Controller Backend

A lightweight, self-hosted game server management platform written in Go, designed for private communities and friends.

## Features

-   **Game Server Management**:
    -   Create, start, stop, and configure **Minecraft** (Vanilla/Fabric) servers.
    -   Real-time console access with command execution.
    -   Server resource monitoring (CPU/RAM).
    -   Configurable `server.properties` editor.
    -   Backup management (World data).
    -   *Experimental support for Among Us servers.*
-   **User System**:
    -   Private access control (No public registration).
    -   Role-based permissions (Root/Admin/User).
    -   Secure authentication via JWT & Cookies.
    -   Multi-Factor Authentication (Email Verification) for new devices.
-   **Architecture**:
    -   **Single Binary Deployment**: Frontend (Vue 3) is embedded into the Go binary.
    -   **Lightweight**: Uses SQLite for data storage, no heavy database required.
    -   **Self-Contained**: Environment-based configuration.

## Getting Started

### Prerequisites

-   Go 1.24+
-   Node.js 18+ (for building frontend)
-   Java (JDK 17/21 recommended) installed and available in system PATH (for running Minecraft servers).

### Installation

1.  **Clone the repository**
    ```bash
    git clone https://github.com/carsupper665/mc-server-backend.git
    cd mc-server-backend
    ```

2.  **Build Frontend & Backend**
    This command will build the Vue frontend and embed it into the Go binary.
    ```bash
    # Windows
    cd frontend && npm install && npm run build
    cd ..
    go build -o server.exe main.go

    # Linux/Mac
    cd frontend && npm install && npm run build
    cd ..
    go build -o server main.go
    ```

3.  **Configuration**
    Create a `.env` file in the root directory (see configuration below).

4.  **Run**
    ```bash
    # Windows
    ./server.exe
    
    # Linux/Mac
    ./server
    ```
    Access the dashboard at `http://localhost:8080`.

    **注意**：前端静态文件已经嵌入到 Go 二进制文件中，运行服务器后会自动提供前端服务。无需单独启动前端服务器。

### 开发模式（前端热重载）

如果你正在开发前端代码，可以使用开发模式以获得热重载功能：

1. **启动后端服务器**（在一个终端窗口）
   ```bash
   go run main.go
   # 或
   ./server.exe
   ```

2. **启动前端开发服务器**（在另一个终端窗口）
   ```bash
   cd frontend
   npm install  # 如果还没安装依赖
   npm run dev
   ```

3. **访问前端**
   - 前端开发服务器通常运行在 `http://localhost:5173`（Vite 默认端口）
   - 前端会自动将 API 请求代理到 `http://localhost:8080`（后端）
   - 修改前端代码后会自动刷新，无需重新构建

**开发模式的优势**：
- ✅ 热重载：修改代码立即看到效果
- ✅ 更快的编译速度
- ✅ 更好的错误提示
- ✅ 支持 Vue DevTools 调试

### 单独测试前端静态文件（开发调试用）

如果你想单独测试编译后的前端静态文件（不通过 Go 后端），可以使用以下方法：

1. **使用 Vite Preview（推荐）**
   ```bash
   cd frontend
   npm run preview
   ```
   这会在 `http://localhost:4173` 启动一个预览服务器。

2. **使用 Python HTTP 服务器**
   ```bash
   # Python 3
   cd frontend/dist
   python -m http.server 8080
   
   # Python 2
   python -m SimpleHTTPServer 8080
   ```

3. **使用 Node.js serve**
   ```bash
   npx serve frontend/dist -p 8080
   ```

**注意**：单独运行静态文件时，API 请求会失败（因为后端未运行）。仅用于测试前端界面，完整功能需要运行 Go 服务器。

---

## `.env` Configuration

Create a file named `.env` in the project root:

```dotenv
# Server Configuration
PORT=8080
DEBUG=false

# Security (Change these!)
SESSION_SECRET=change-this-to-a-random-string
CRYPTO_SECRET=change-this-to-another-random-string

# Database
SQLITE_PATH=./data/db.sqlite
SQL_MAX_IDLE_CONNS=100
SQL_MAX_OPEN_CONNS=1000
SQL_MAX_LIFETIME=60

# Initial Admin Setup
CREATE_ROOT_USER=true
ROOT_USER_NAME=admin
ROOT_USER_EMAIL=admin@example.com
ROOT_USER_PASSWORD=securepassword

# Game Server Settings
MINECRAFT_SERVER_PATH=./minecraft_servers
LATEST_FABRIC_INSTALLER_VERSION=1.1.0

# Email (Optional, for MFA verification code)
SMTP_SERVER=smtp.example.com
SMTP_PORT=587
SMTP_ACCOUNT=noreply@example.com
SMTP_TOKEN=your-smtp-password
SMTP_FROM=noreply@example.com
SMTP_SSL_ENABLED=false

# System Tuning
GLOBAL_API_RATE_LIMIT=60
GLOBAL_API_RATE_LIMIT_DURATION=60
UA_FILTER=true
```

## Project Structure

```
├── common/         # Shared utilities, constants, config loader
├── controller/     # HTTP Request handlers
├── frontend/       # Vue 3 + Vite Frontend source
├── middleware/     # Gin middlewares (Auth, CORS, Logging)
├── model/          # Database models (SQLite/GORM)
├── router/         # API Routes & Static file serving
├── service/        # Business logic (Server process management)
└── main.go         # Entry point
```

## Tech Stack

-   **Backend**: Go (Gin Framework), GORM (SQLite), Gorilla WebSocket
-   **Frontend**: Vue 3, TypeScript, Vite, Tailwind CSS, Pinia
-   **Infrastructure**: Embed (Go 1.16+), Viper (Config)

## License

This project is created for personal use and experimentation.
