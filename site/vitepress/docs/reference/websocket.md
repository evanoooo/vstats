# WebSocket API

vStats uses WebSocket for real-time metric updates between agents and the dashboard.

## Connection

### Dashboard Client

Connect to receive real-time updates:

```
ws://your-server:3001/ws
```

### Agent Connection

Agents connect with authentication:

```
ws://your-server:3001/ws/agent?token=<agent-token>
```

## Message Format

All messages are JSON formatted:

```json
{
  "type": "message_type",
  "data": { ... },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Client Messages

Messages that clients (dashboard) can receive:

### Server Metrics Update

```json
{
  "type": "metrics",
  "data": {
    "server_id": "abc123",
    "metrics": {
      "cpu": { "usage": 45.2, "cores": 4 },
      "memory": { "total": 8589934592, "used": 4294967296 },
      "disks": [...],
      "network": { "rx": 1024, "tx": 2048 }
    }
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Server Status Change

```json
{
  "type": "status",
  "data": {
    "server_id": "abc123",
    "online": true
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Server Added/Removed

```json
{
  "type": "server_added",
  "data": {
    "id": "abc123",
    "name": "new-server",
    "location": "US-West"
  }
}
```

```json
{
  "type": "server_removed",
  "data": {
    "id": "abc123"
  }
}
```

## Agent Messages

Messages that agents send to the server:

### Metrics Report

```json
{
  "type": "metrics",
  "data": {
    "timestamp": "2024-01-01T12:00:00Z",
    "hostname": "my-server",
    "os": {
      "name": "Ubuntu",
      "version": "22.04",
      "kernel": "5.15.0",
      "arch": "x86_64"
    },
    "cpu": {
      "brand": "Intel Core i7",
      "cores": 8,
      "usage": 45.2,
      "frequency": 3600,
      "per_core": [40, 50, 45, 42, 48, 46, 44, 47]
    },
    "memory": {
      "total": 17179869184,
      "used": 8589934592,
      "available": 8589934592,
      "swap_total": 4294967296,
      "swap_used": 1073741824,
      "usage_percent": 50.0
    },
    "disks": [
      {
        "name": "sda1",
        "mount_point": "/",
        "fs_type": "ext4",
        "total": 107374182400,
        "used": 53687091200,
        "available": 53687091200,
        "usage_percent": 50.0
      }
    ],
    "network": {
      "interfaces": [
        { "name": "eth0", "rx": 1073741824, "tx": 536870912 }
      ],
      "total_rx": 1073741824,
      "total_tx": 536870912
    },
    "uptime": 86400,
    "load_average": {
      "one": 0.5,
      "five": 0.7,
      "fifteen": 0.6
    }
  }
}
```

### Heartbeat

```json
{
  "type": "heartbeat",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Connection Handling

### Reconnection

Clients should implement automatic reconnection with exponential backoff:

```javascript
class WebSocketClient {
  constructor(url) {
    this.url = url;
    this.reconnectDelay = 1000;
    this.maxDelay = 30000;
    this.connect();
  }

  connect() {
    this.ws = new WebSocket(this.url);
    
    this.ws.onopen = () => {
      this.reconnectDelay = 1000;
      console.log('Connected');
    };

    this.ws.onclose = () => {
      console.log('Disconnected, reconnecting...');
      setTimeout(() => this.connect(), this.reconnectDelay);
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxDelay);
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };
  }

  handleMessage(message) {
    switch (message.type) {
      case 'metrics':
        // Handle metrics update
        break;
      case 'status':
        // Handle status change
        break;
    }
  }
}
```

### Ping/Pong

The server sends ping frames every 30 seconds. Clients must respond with pong to maintain the connection.

Most WebSocket libraries handle this automatically.

## Browser Example

```javascript
const ws = new WebSocket('ws://localhost:3001/ws');

ws.onopen = () => {
  console.log('Connected to vStats');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  if (message.type === 'metrics') {
    const { server_id, metrics } = message.data;
    console.log(`Server ${server_id}: CPU ${metrics.cpu.usage}%`);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected');
};
```

## Security

- Always use `wss://` in production (WebSocket over TLS)
- Agent connections require valid tokens
- Invalid tokens result in immediate disconnection
- Connections from unknown origins may be rejected

