package main

import (
	"vstats/internal/common"
)

// Re-export common types for convenience
type SystemMetrics = common.SystemMetrics
type OsInfo = common.OsInfo
type CpuMetrics = common.CpuMetrics
type MemoryMetrics = common.MemoryMetrics
type MemoryModule = common.MemoryModule
type DiskMetrics = common.DiskMetrics
type NetworkMetrics = common.NetworkMetrics
type NetworkInterface = common.NetworkInterface
type LoadAverage = common.LoadAverage
type PingMetrics = common.PingMetrics
type PingTarget = common.PingTarget
type PingTargetConfig = common.PingTargetConfig
type AuthMessage = common.AuthMessage
type MetricsMessage = common.MetricsMessage
type ServerResponse = common.ServerResponse
type RegisterRequest = common.RegisterRequest
type RegisterResponse = common.RegisterResponse

// Batch metrics types for offline sync
type BatchMetricsMessage = common.BatchMetricsMessage
type TimestampedMetrics = common.TimestampedMetrics
type CommonAggregatedMetrics = common.AggregatedMetrics
type BatchMetricsResponse = common.BatchMetricsResponse
