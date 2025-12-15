# WebSocket API

vStats 使用 WebSocket 在 Agent 和面板之间进行实时指标更新。

## 连接

### 面板客户端

连接以接收实时更新：

```
ws://your-server:3001/ws
```

### Agent 连接

Agent 带认证连接：

```
ws://your-server:3001/ws/agent?token=<agent-token>
```

## 消息格式

所有消息为 JSON 格式：

```json
{
  "type": "message_type",
  "data": { ... },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## 客户端消息

客户端（面板）可以接收的消息：

### 服务器指标更新

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

### 服务器状态变化

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

### 服务器添加/删除

```json
{
  "type": "server_added",
  "data": {
    "id": "abc123",
    "name": "new-server",
    "location": "上海"
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

## Agent 消息

Agent 发送到服务端的消息：

### 指标报告

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
      "usage": 45.2
    },
    "memory": {
      "total": 17179869184,
      "used": 8589934592,
      "usage_percent": 50.0
    },
    "disks": [...],
    "network": {...},
    "uptime": 86400,
    "load_average": {
      "one": 0.5,
      "five": 0.7,
      "fifteen": 0.6
    }
  }
}
```

### 心跳

```json
{
  "type": "heartbeat",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## 连接处理

### 重连

客户端应实现带指数退避的自动重连：

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
      console.log('已连接');
    };

    this.ws.onclose = () => {
      console.log('断开连接，正在重连...');
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
        // 处理指标更新
        break;
      case 'status':
        // 处理状态变化
        break;
    }
  }
}
```

### Ping/Pong

服务器每 30 秒发送 ping 帧。客户端必须响应 pong 以维持连接。

大多数 WebSocket 库会自动处理这个。

## 浏览器示例

```javascript
const ws = new WebSocket('ws://localhost:3001/ws');

ws.onopen = () => {
  console.log('已连接到 vStats');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  if (message.type === 'metrics') {
    const { server_id, metrics } = message.data;
    console.log(`服务器 ${server_id}: CPU ${metrics.cpu.usage}%`);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error);
};

ws.onclose = () => {
  console.log('已断开');
};
```

## 安全

- 生产环境始终使用 `wss://`（WebSocket over TLS）
- Agent 连接需要有效的 token
- 无效的 token 会导致立即断开
- 来自未知来源的连接可能被拒绝

