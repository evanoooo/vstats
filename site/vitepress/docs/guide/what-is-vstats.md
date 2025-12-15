# What is vStats?

vStats is a minimalist and beautiful server monitoring dashboard. Built with Go and React, it provides real-time system metrics with millisecond-level latency.

## Features

### ⚡ Real-time Monitoring
- WebSocket-powered real-time data push
- Millisecond-level latency
- Live charts and graphs

### 🖥️ Multi-Server Management
- Monitor multiple servers from one dashboard
- Unified server list view
- Individual server detail pages

### 📊 Comprehensive Metrics
- **CPU**: Usage, cores, frequency, per-core stats
- **Memory**: Total, used, available, swap
- **Disk**: Multiple disk support, I/O stats
- **Network**: Bandwidth, interfaces, total traffic
- **GPU**: NVIDIA GPU support (optional)
- **System**: Uptime, load average, OS info

### 🎨 Modern UI
- Glassmorphism design
- Smooth animations
- Multiple theme options
- Responsive layout

### 🔐 Secure
- JWT-based authentication
- Admin-only operations
- Secure WebSocket connections

### 🚀 Easy Deployment
- Docker one-liner
- Script installation
- Systemd service management

## Architecture

vStats consists of two main components:

```
┌─────────────────────────────────────────────────────────────┐
│                    vStats Dashboard                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Web UI    │  │  REST API   │  │  WebSocket  │          │
│  │   (React)   │  │    (Gin)    │  │  (Gorilla)  │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│                          │                                   │
│              ┌───────────┴───────────┐                       │
│              │      Go Backend       │                       │
│              │    + SQLite DB        │                       │
│              └───────────────────────┘                       │
└─────────────────────────────────────────────────────────────┘
                           │ WebSocket
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │   Agent 1   │ │   Agent 2   │ │   Agent N   │
    │   (Linux)   │ │  (Windows)  │ │   (macOS)   │
    └─────────────┘ └─────────────┘ └─────────────┘
```

### Dashboard (Server)
The main component that:
- Serves the web UI
- Provides REST API
- Manages WebSocket connections
- Stores server data in SQLite

### Agent
Lightweight agent that:
- Collects system metrics
- Sends data via WebSocket
- Runs as a system service
- Supports Linux, Windows, macOS

## Technology Stack

| Component | Technology |
|-----------|------------|
| Backend | Go (Gin, Gorilla WebSocket) |
| Frontend | React, TypeScript, Tailwind CSS |
| Database | SQLite |
| Deployment | Docker, Systemd |

## Requirements

### Server
- Linux, macOS, or Windows
- 512MB RAM minimum
- 100MB disk space
- Port 3001 (configurable)

### Agent
- Linux, macOS, or Windows
- 64MB RAM
- Network access to server

## Next Steps

- [Getting Started](/guide/getting-started) - Quick start guide
- [Docker Installation](/guide/installation/docker) - Deploy with Docker
- [Agent Installation](/guide/agent/install) - Install monitoring agent

