# Architecture

This document describes the overall architecture of vStats.

## System Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              Users                                        │
│                    (Browser / Mobile / API Clients)                       │
└─────────────────────────────────┬────────────────────────────────────────┘
                                  │ HTTPS / WSS
                                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                         Reverse Proxy (Optional)                          │
│                      (Nginx / Caddy / Traefik)                           │
└─────────────────────────────────┬────────────────────────────────────────┘
                                  │ HTTP / WS
                                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           vStats Server                                   │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                         Go Backend                                  │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │  │
│  │  │  HTTP Server │  │   WebSocket  │  │   Background Workers     │  │  │
│  │  │    (Gin)     │  │   Handler    │  │  (Aggregation, Cleanup)  │  │  │
│  │  └──────┬───────┘  └──────┬───────┘  └────────────┬─────────────┘  │  │
│  │         │                 │                       │                 │  │
│  │         └─────────────────┴───────────────────────┘                 │  │
│  │                           │                                         │  │
│  │  ┌────────────────────────┴────────────────────────────────────┐   │  │
│  │  │                    Data Layer                                │   │  │
│  │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │   │  │
│  │  │  │   SQLite DB  │  │  In-Memory   │  │   File Storage   │   │   │  │
│  │  │  │   (Primary)  │  │    Cache     │  │     (Logs)       │   │   │  │
│  │  │  └──────────────┘  └──────────────┘  └──────────────────┘   │   │  │
│  │  └─────────────────────────────────────────────────────────────┘   │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                     Static File Server                              │  │
│  │                    (React SPA + Assets)                             │  │
│  └────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────┘
                                  │ WebSocket
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
          ▼                       ▼                       ▼
   ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
   │   Agent 1   │         │   Agent 2   │         │   Agent N   │
   │   (Linux)   │         │  (Windows)  │         │   (macOS)   │
   └─────────────┘         └─────────────┘         └─────────────┘
```

## Components

### vStats Server

The central component that:

1. **HTTP Server (Gin)**
   - Serves REST API endpoints
   - Handles authentication
   - Serves static frontend files

2. **WebSocket Handler**
   - Maintains connections with agents
   - Broadcasts updates to dashboard clients
   - Handles real-time communication

3. **Background Workers**
   - Aggregates metrics for historical data
   - Cleans up old data
   - Monitors agent health

4. **Data Layer**
   - SQLite for persistent storage
   - In-memory cache for real-time data
   - File system for logs

### vStats Agent

Lightweight daemon running on monitored servers:

1. **Metric Collectors**
   - CPU stats from `/proc/stat` (Linux) or system calls
   - Memory from `/proc/meminfo` or system APIs
   - Disk from filesystem stats
   - Network from `/proc/net/dev` or system APIs
   - GPU from `nvidia-smi` (if available)

2. **WebSocket Client**
   - Maintains persistent connection to server
   - Sends metrics at configured intervals
   - Handles reconnection automatically

3. **Configuration Manager**
   - Reads config from file or environment
   - Supports runtime configuration updates

### Frontend (React SPA)

Single-page application built with:

- **React 18** - UI framework
- **TypeScript** - Type safety
- **Tailwind CSS** - Styling
- **Framer Motion** - Animations
- **Recharts** - Data visualization

## Data Flow

### Metric Collection

```
Agent                   Server                  Dashboard
  │                        │                        │
  │ collect metrics        │                        │
  │◄──────────────────────►│                        │
  │                        │                        │
  │ send via WebSocket     │                        │
  │───────────────────────►│                        │
  │                        │                        │
  │                        │ store in DB            │
  │                        │◄──────────────────────►│
  │                        │                        │
  │                        │ broadcast to clients   │
  │                        │───────────────────────►│
  │                        │                        │
  │                        │                        │ update UI
  │                        │                        │◄─────────
```

### Authentication Flow

```
User                    Server                  Database
  │                        │                        │
  │ POST /api/auth/login   │                        │
  │───────────────────────►│                        │
  │                        │                        │
  │                        │ verify credentials     │
  │                        │───────────────────────►│
  │                        │                        │
  │                        │◄───────────────────────│
  │                        │                        │
  │ receive JWT token      │                        │
  │◄───────────────────────│                        │
  │                        │                        │
  │ subsequent requests    │                        │
  │ with Authorization     │                        │
  │───────────────────────►│                        │
  │                        │                        │
  │                        │ verify token           │
  │                        │ (local, no DB)         │
  │                        │                        │
```

## Database Schema

### Servers Table

```sql
CREATE TABLE servers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  location TEXT,
  provider TEXT,
  token TEXT UNIQUE,
  online BOOLEAN DEFAULT FALSE,
  last_seen DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Metrics Table

```sql
CREATE TABLE metrics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  server_id TEXT NOT NULL,
  timestamp DATETIME NOT NULL,
  cpu_usage REAL,
  memory_usage REAL,
  disk_usage TEXT,  -- JSON
  network_rx INTEGER,
  network_tx INTEGER,
  FOREIGN KEY (server_id) REFERENCES servers(id)
);

CREATE INDEX idx_metrics_server_time ON metrics(server_id, timestamp);
```

### Settings Table

```sql
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Scalability Considerations

### Current Limitations

- Single server instance
- SQLite database (single-writer)
- In-memory metric cache

### Future Scaling Options

1. **Horizontal Scaling**
   - Add PostgreSQL/MySQL support
   - Use Redis for shared cache
   - Load balancer for multiple server instances

2. **Metric Storage**
   - Time-series database (InfluxDB, TimescaleDB)
   - Separate hot/cold storage

3. **Agent Scale**
   - Agent proxy/aggregator nodes
   - Regional servers

## Security Model

### Authentication

- JWT tokens for API access
- Token expiration and refresh
- Secure password hashing (bcrypt)

### Agent Authentication

- Pre-shared tokens
- TLS encryption (in production)
- Token rotation support

### Network Security

- CORS configuration
- Rate limiting
- Input validation

## Technology Choices

| Component | Technology | Reason |
|-----------|------------|--------|
| Backend Language | Go | Performance, single binary, concurrency |
| HTTP Framework | Gin | Fast, well-maintained |
| WebSocket | Gorilla | Robust, full-featured |
| Database | SQLite | Simple, embedded, sufficient for most deployments |
| Frontend | React | Component-based, large ecosystem |
| Styling | Tailwind | Utility-first, fast development |
| Build Tool | Vite | Fast builds, modern tooling |

