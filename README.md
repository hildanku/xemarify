# Xemarify
## Getting Started (Production)

### Requirements

- 1 VM for Manager + Web
- 1+ VM for Agent
- Ubuntu 22.04 LTS

### Step 1 - Install Manager

On your manager VM, run:

```bash
curl -fsSL https://raw.githubusercontent.com/hildanku/xemarify/main/deploy/manager/install.sh | sudo bash
```

To pin a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/hildanku/xemarify/main/deploy/manager/install.sh | \
  sudo MANAGER_VERSION=v1.1.0-beta WEB_VERSION=v1.1.0-beta bash
```

The installer will:
1. Install Docker (if not present)
2. Pull the Manager and Web images from GHCR
3. Generate secrets (DB password, JWT, setup token)
4. Start the full stack (Postgres, Manager, Web)
5. Print the dashboard URL and a one-time **setup token**

### Step 2 - Create Your Account

Open the dashboard URL printed by the installer (default `http://<your-ip>:3000`) and use the setup token to create your first manager account.

### Step 3 - Install Agent

On each host you want to monitor, run:

```bash
curl -fsSL https://raw.githubusercontent.com/hildanku/xemarify/main/deploy/agent/install.sh | \
  sudo MANAGER_ENDPOINT=http://<manager-ip>:8089 ENROLLMENT_TOKEN=<token> bash
```

Get the enrollment token from the Manager dashboard (Agents > Enroll New Agent).

To pin a specific agent version:

```bash
curl -fsSL https://raw.githubusercontent.com/hildanku/xemarify/main/deploy/agent/install.sh | \
  sudo MANAGER_ENDPOINT=http://<manager-ip>:8089 ENROLLMENT_TOKEN=<token> VERSION=v1.1.0-beta bash
```

The agent will automatically:
1. Download the binary from GitHub Releases
2. Write config to `/etc/xemarify-agent/agent.yaml`
3. Install and start a systemd service
4. Register itself with the Manager

### Verify

```bash
# On manager VM
cd /opt/xemarify && docker compose ps

# On agent VM
systemctl status xemarify-agent
```

---

## Development Setup

### Prerequisites

- Go 1.23+ (agent) / Go 1.25+ (manager)
- Node.js 22+
- Docker & Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

### 1. Start the Database

```bash
docker compose up -d
```

This starts PostgreSQL on `localhost:5445`.

### 2. Start the Manager

```bash
cd manager
cp .env.example .env    # edit JWT_SECRET and MANAGER_SETUP_TOKEN
make migrate-up         # apply database migrations
make seed               # (optional) create default users
make dev                # start the API server on :8089
```

Default seed users:

| Username | Email | Role | Password |
|----------|-------|------|----------|
| manager | manager@xemarify.local | MANAGER | Manager@123 |
| analyst | analyst@xemarify.local | ANALYST | Analyst@123 |
| viewer | viewer@xemarify.local | VIEWER | Viewer@123 |

### 3. Start the Web Dashboard

```bash
cd web
npm install
npm run dev             # starts on http://localhost:5173
```

The dashboard connects to the Manager API at `http://localhost:8089` by default (configured via `VITE_API_BASE_URL`).

### 4. (Optional) Start the Agent

```bash
cd agent
sudo mkdir -p /etc/xemarify-agent /var/lib/xemarify-agent/spool /var/log/xemarify-agent
sudo cp example.yaml /etc/xemarify-agent/agent.yaml
# Edit agent.yaml: set enrollment_token from the Manager dashboard
go run main.go
```

### Dev Ports

| Service | Port |
|---------|------|
| PostgreSQL | 5445 |
| Manager API | 8089 |
| Web (Vite) | 5173 |
| Agent Syslog | 5514 (UDP) |

### Useful Commands

```bash
# Manager
cd manager
make build              # build binary to bin/manager
make migrate-down       # rollback 1 migration
make db-reset           # drop all + re-migrate

# Web
cd web
npm run check           # type-check
npm run lint            # prettier + eslint
npm run test            # run unit tests

# Agent
cd agent
go build -o xemarify-agent .
```

---

## Architecture

```
┌─────────────┐         ┌─────────────────┐
│   Agent     │────────▶│    Manager      │
│  (per host) │  events │  (Go API)       │
└─────────────┘  + hb   │                 │
                         │  - Detection    │
┌─────────────┐         │  - Alerting     │
│   Agent     │────────▶│  - SSE Stream   │
└─────────────┘         └────────┬────────┘
                                 │
                         ┌───────▼────────┐
                         │   Web Dashboard │
                         │   (SvelteKit)   │
                         └────────────────┘
```

---

## License

[AGPL-3.0](LICENSE)
