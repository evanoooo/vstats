# vStats Server (Go Implementation)

这是 vStats 服务器的 Go 语言实现。

## 构建

```bash
cd server-go
go mod tidy
go build -o vstats-server
```

## 运行

```bash
./vstats-server
```

## 命令行选项

- `--check`: 显示诊断信息
- `--reset-password`: 重置管理员密码

## 环境变量

- `VSTATS_PORT`: 服务器端口（默认: 3001）

## API 端点

- `GET /health` - 健康检查
- `GET /api/metrics` - 获取本地服务器指标
- `GET /api/metrics/all` - 获取所有服务器指标
- `GET /api/history/:server_id?range=1h|24h|7d|30d` - 获取历史数据
- `POST /api/auth/login` - 登录
- `GET /api/auth/verify` - 验证令牌
- `GET /ws` - Dashboard WebSocket
- `GET /ws/agent` - Agent WebSocket

## 配置文件

配置文件位置：与可执行文件同目录下的 `vstats-config.json`

## 数据库

SQLite 数据库位置：与可执行文件同目录下的 `vstats.db`

