# vStats Debug Data Generator

生成调试数据用于测试 vStats 仪表板。支持新的多粒度聚合协议。

## 功能

1. **历史数据生成** - 生成多粒度聚合历史数据（5sec/2min/15min/hourly/daily）
2. **实时 Agent 模拟** - 启动多个模拟 Agent 实时向 Server 汇报数据

## 新协议特性

- **多粒度聚合**: 发送预聚合的 bucket 数据，模拟真实 agent 行为
- **5种时间粒度**: 5sec, 2min, 15min, hourly, daily
- **Ping 聚合**: 每个目标的延迟统计（sum/max/count/ok/fail）
- **实时 + 聚合**: 同时发送实时指标和定期聚合数据

## 快速开始

```bash
# 确保 vStats server 正在运行
cd scripts/debug-data

# 运行（生成历史数据 + 启动实时 agents）
# 会提示输入 admin token（隐藏输入）
./run.sh

# 仅生成 2 小时历史数据
./run.sh history

# 仅运行实时 agents
./run.sh realtime

# 清理所有调试服务器
./run.sh cleanup
```

## 配置选项

通过环境变量配置：

```bash
# Server URL（默认 http://localhost:3001）
export VSTATS_SERVER_URL=http://localhost:3001

# 模拟服务器数量（默认 100）
export VSTATS_SERVER_COUNT=100

# 历史数据时长（默认 2 小时）
export VSTATS_HISTORY_HOURS=2

# 实时汇报间隔（默认 3 秒）
export VSTATS_INTERVAL=3

# 聚合数据同步间隔（默认 60 秒）
export VSTATS_AGG_INTERVAL=60
```

或直接使用命令行参数：

```bash
# 通过命令行传递 token（不推荐，会留在历史记录中）
go run main.go \
    --server http://localhost:3001 \
    --count 100 \
    --hours 2 \
    --interval 3 \
    --agg-interval 60 \
    --token "your-admin-token" \
    --mode both

# 不传 token 会提示安全输入
go run main.go --count 50 --mode realtime
```

## 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--server` | http://localhost:3001 | vStats Server 地址 |
| `--count` | 100 | 模拟服务器数量 |
| `--hours` | 2 | 生成历史数据的小时数 |
| `--interval` | 3 | 实时汇报间隔（秒） |
| `--agg-interval` | 60 | 聚合数据同步间隔（秒） |
| `--token` | (提示输入) | Admin Token（用于 API 认证） |
| `--mode` | both | 运行模式：history/realtime/both |
| `--cleanup` | false | 仅清理调试服务器后退出 |

## 工作原理

### 历史数据模式
1. 通过 HTTP API (`/api/servers`) 注册调试服务器
2. 通过 WebSocket 连接到 `/ws/agent` 端点并认证
3. 发送 `aggregated_metrics` 消息，包含所有粒度的历史 bucket 数据
4. 数据直接入库到 `metrics_*_agg` 和 `ping_*_agg` 表

### 实时模式
1. 每 `--interval` 秒发送 `metrics` 消息（实时指标）
2. 每 `--agg-interval` 秒发送 `aggregated_metrics` 消息（聚合数据）
3. 模拟真实 agent 的双轨数据上报行为

## 清理调试数据

```bash
# 自动清理所有 Debug-Server-* 服务器
./run.sh cleanup

# 或在仪表板中手动删除
```

## 注意事项

- 确保 vStats Server 正在运行（v3.2.13+，支持新协议）
- 需要 Admin Token（从仪表板设置页面获取）
- 首次运行时会下载 Go 依赖
- 实时模式下按 `Ctrl+C` 停止所有 agents
- 每次运行会自动清理之前创建的调试服务器
