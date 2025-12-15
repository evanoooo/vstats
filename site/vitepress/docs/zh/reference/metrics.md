# 指标结构

本文档描述 vStats Agent 收集的指标结构。

## 完整指标对象

```typescript
interface SystemMetrics {
  timestamp: string;        // ISO 8601 时间戳
  hostname: string;         // 服务器主机名
  os: OSInfo;
  cpu: CPUInfo;
  memory: MemoryInfo;
  disks: DiskInfo[];
  network: NetworkInfo;
  uptime: number;           // 启动以来的秒数
  load_average: LoadAverage;
  gpu?: GPUInfo[];          // 可选，仅 NVIDIA
}
```

## 操作系统信息

```typescript
interface OSInfo {
  name: string;         // 如 "Ubuntu", "Windows", "macOS"
  version: string;      // 如 "22.04", "11", "14.0"
  kernel: string;       // 如 "5.15.0-generic"
  arch: string;         // 如 "x86_64", "aarch64"
}
```

## CPU 信息

```typescript
interface CPUInfo {
  brand: string;        // 如 "Intel Core i7-12700K"
  cores: number;        // 物理核 + 逻辑核
  usage: number;        // 总体使用率 (0-100)
  frequency: number;    // 当前频率 MHz
  per_core: number[];   // 每核使用率
}
```

## 内存信息

```typescript
interface MemoryInfo {
  total: number;            // 总内存（字节）
  used: number;             // 已用内存（字节）
  available: number;        // 可用内存（字节）
  swap_total: number;       // 总交换分区（字节）
  swap_used: number;        // 已用交换分区（字节）
  usage_percent: number;    // 内存使用率 (0-100)
}
```

## 磁盘信息

```typescript
interface DiskInfo {
  name: string;             // 设备名，如 "sda1"
  mount_point: string;      // 挂载路径，如 "/"
  fs_type: string;          // 文件系统类型，如 "ext4"
  total: number;            // 总空间（字节）
  used: number;             // 已用空间（字节）
  available: number;        // 可用空间（字节）
  usage_percent: number;    // 使用率 (0-100)
}
```

## 网络信息

```typescript
interface NetworkInfo {
  interfaces: NetworkInterface[];
  total_rx: number;     // 总接收字节（自启动）
  total_tx: number;     // 总发送字节（自启动）
}

interface NetworkInterface {
  name: string;         // 接口名，如 "eth0"
  rx: number;           // 接收字节
  tx: number;           // 发送字节
}
```

## 负载平均

```typescript
interface LoadAverage {
  one: number;          // 1 分钟负载平均
  five: number;         // 5 分钟负载平均
  fifteen: number;      // 15 分钟负载平均
}
```

## GPU 信息（可选）

```typescript
interface GPUInfo {
  index: number;            // GPU 索引
  name: string;             // 如 "NVIDIA GeForce RTX 3080"
  temperature: number;      // 温度（摄氏度）
  utilization: number;      // GPU 使用率
  memory_total: number;     // 总显存（字节）
  memory_used: number;      // 已用显存（字节）
  memory_free: number;      // 可用显存（字节）
  power_draw: number;       // 当前功耗（瓦）
  power_limit: number;      // 功耗限制（瓦）
}
```

## 示例载荷

```json
{
  "timestamp": "2024-01-15T12:30:00Z",
  "hostname": "web-server-01",
  "os": {
    "name": "Ubuntu",
    "version": "22.04.3 LTS",
    "kernel": "5.15.0-91-generic",
    "arch": "x86_64"
  },
  "cpu": {
    "brand": "Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz",
    "cores": 8,
    "usage": 45.5,
    "frequency": 2400,
    "per_core": [42.1, 48.3, 45.0, 43.2, 47.8, 44.5, 46.1, 45.0]
  },
  "memory": {
    "total": 17179869184,
    "used": 8589934592,
    "available": 8589934592,
    "swap_total": 4294967296,
    "swap_used": 536870912,
    "usage_percent": 50.0
  },
  "disks": [
    {
      "name": "sda1",
      "mount_point": "/",
      "fs_type": "ext4",
      "total": 107374182400,
      "used": 42949672960,
      "available": 64424509440,
      "usage_percent": 40.0
    }
  ],
  "network": {
    "interfaces": [
      {
        "name": "eth0",
        "rx": 1073741824000,
        "tx": 536870912000
      }
    ],
    "total_rx": 1073741824000,
    "total_tx": 536870912000
  },
  "uptime": 2592000,
  "load_average": {
    "one": 1.25,
    "five": 1.10,
    "fifteen": 0.95
  }
}
```

## 单位参考

| 指标 | 单位 |
|------|------|
| 内存/磁盘大小 | 字节 |
| CPU 频率 | MHz |
| 温度 | 摄氏度 |
| 功耗 | 瓦 |
| 运行时间 | 秒 |
| 使用率 | 百分比 (0-100) |
| 网络流量 | 字节（累计） |

## 采集间隔

默认情况下，指标每 1 秒采集一次。可在 Agent 设置中配置。

## 数据保留

历史指标存储时间：
- 原始数据：24 小时
- 1 分钟聚合：7 天
- 1 小时聚合：30 天
- 1 天聚合：1 年

可在服务器设置中配置保留期。

