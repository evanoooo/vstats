# Metrics Schema

This document describes the structure of metrics collected by vStats agents.

## Complete Metrics Object

```typescript
interface SystemMetrics {
  timestamp: string;        // ISO 8601 timestamp
  hostname: string;         // Server hostname
  os: OSInfo;
  cpu: CPUInfo;
  memory: MemoryInfo;
  disks: DiskInfo[];
  network: NetworkInfo;
  uptime: number;           // Seconds since boot
  load_average: LoadAverage;
  gpu?: GPUInfo[];          // Optional, NVIDIA only
}
```

## OS Information

```typescript
interface OSInfo {
  name: string;         // e.g., "Ubuntu", "Windows", "macOS"
  version: string;      // e.g., "22.04", "11", "14.0"
  kernel: string;       // e.g., "5.15.0-generic"
  arch: string;         // e.g., "x86_64", "aarch64"
}
```

## CPU Information

```typescript
interface CPUInfo {
  brand: string;        // e.g., "Intel Core i7-12700K"
  cores: number;        // Physical + logical cores
  usage: number;        // Overall usage percentage (0-100)
  frequency: number;    // Current frequency in MHz
  per_core: number[];   // Per-core usage percentages
}
```

## Memory Information

```typescript
interface MemoryInfo {
  total: number;            // Total RAM in bytes
  used: number;             // Used RAM in bytes
  available: number;        // Available RAM in bytes
  swap_total: number;       // Total swap in bytes
  swap_used: number;        // Used swap in bytes
  usage_percent: number;    // RAM usage percentage (0-100)
}
```

## Disk Information

```typescript
interface DiskInfo {
  name: string;             // Device name, e.g., "sda1"
  mount_point: string;      // Mount path, e.g., "/"
  fs_type: string;          // Filesystem type, e.g., "ext4"
  total: number;            // Total space in bytes
  used: number;             // Used space in bytes
  available: number;        // Available space in bytes
  usage_percent: number;    // Usage percentage (0-100)
}
```

## Network Information

```typescript
interface NetworkInfo {
  interfaces: NetworkInterface[];
  total_rx: number;     // Total received bytes (since boot)
  total_tx: number;     // Total transmitted bytes (since boot)
}

interface NetworkInterface {
  name: string;         // Interface name, e.g., "eth0"
  rx: number;           // Received bytes
  tx: number;           // Transmitted bytes
}
```

## Load Average

```typescript
interface LoadAverage {
  one: number;          // 1-minute load average
  five: number;         // 5-minute load average
  fifteen: number;      // 15-minute load average
}
```

## GPU Information (Optional)

```typescript
interface GPUInfo {
  index: number;            // GPU index
  name: string;             // e.g., "NVIDIA GeForce RTX 3080"
  temperature: number;      // Temperature in Celsius
  utilization: number;      // GPU utilization percentage
  memory_total: number;     // Total VRAM in bytes
  memory_used: number;      // Used VRAM in bytes
  memory_free: number;      // Free VRAM in bytes
  power_draw: number;       // Current power draw in watts
  power_limit: number;      // Power limit in watts
}
```

## Example Payload

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
    },
    {
      "name": "sdb1",
      "mount_point": "/data",
      "fs_type": "xfs",
      "total": 1099511627776,
      "used": 549755813888,
      "available": 549755813888,
      "usage_percent": 50.0
    }
  ],
  "network": {
    "interfaces": [
      {
        "name": "eth0",
        "rx": 1073741824000,
        "tx": 536870912000
      },
      {
        "name": "eth1",
        "rx": 107374182400,
        "tx": 53687091200
      }
    ],
    "total_rx": 1181116006400,
    "total_tx": 590558003200
  },
  "uptime": 2592000,
  "load_average": {
    "one": 1.25,
    "five": 1.10,
    "fifteen": 0.95
  },
  "gpu": [
    {
      "index": 0,
      "name": "NVIDIA GeForce RTX 3080",
      "temperature": 65,
      "utilization": 78,
      "memory_total": 10737418240,
      "memory_used": 8589934592,
      "memory_free": 2147483648,
      "power_draw": 285.5,
      "power_limit": 320.0
    }
  ]
}
```

## Units Reference

| Metric | Unit |
|--------|------|
| Memory/Disk sizes | Bytes |
| CPU frequency | MHz |
| Temperature | Celsius |
| Power | Watts |
| Uptime | Seconds |
| Usage/Utilization | Percentage (0-100) |
| Network traffic | Bytes (cumulative) |

## Collection Interval

By default, metrics are collected every 1 second. This can be configured in the agent settings.

## Data Retention

Historical metrics are stored for:
- Raw data: 24 hours
- 1-minute aggregates: 7 days
- 1-hour aggregates: 30 days
- 1-day aggregates: 1 year

Configure retention in server settings.

