use chrono::{Duration, Utc};
use rusqlite::{Connection, params};

use crate::config::get_db_path;
use crate::types::SystemMetrics;

pub fn init_database() -> rusqlite::Result<Connection> {
    let conn = Connection::open(get_db_path())?;
    
    conn.execute_batch(r#"
        -- Raw metrics (keep for 24 hours)
        CREATE TABLE IF NOT EXISTS metrics_raw (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            server_id TEXT NOT NULL,
            timestamp TEXT NOT NULL,
            cpu_usage REAL NOT NULL,
            memory_usage REAL NOT NULL,
            disk_usage REAL NOT NULL,
            net_rx INTEGER NOT NULL,
            net_tx INTEGER NOT NULL,
            load_1 REAL NOT NULL,
            load_5 REAL NOT NULL,
            load_15 REAL NOT NULL,
            ping_ms REAL,
            created_at TEXT DEFAULT CURRENT_TIMESTAMP
        );
        
        -- Hourly aggregated metrics (keep for 30 days)
        CREATE TABLE IF NOT EXISTS metrics_hourly (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            server_id TEXT NOT NULL,
            hour_start TEXT NOT NULL,
            cpu_avg REAL NOT NULL,
            cpu_max REAL NOT NULL,
            memory_avg REAL NOT NULL,
            memory_max REAL NOT NULL,
            disk_avg REAL NOT NULL,
            net_rx_total INTEGER NOT NULL,
            net_tx_total INTEGER NOT NULL,
            ping_avg REAL,
            sample_count INTEGER NOT NULL,
            UNIQUE(server_id, hour_start)
        );
        
        -- Daily aggregated metrics (keep forever)
        CREATE TABLE IF NOT EXISTS metrics_daily (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            server_id TEXT NOT NULL,
            date TEXT NOT NULL,
            cpu_avg REAL NOT NULL,
            cpu_max REAL NOT NULL,
            memory_avg REAL NOT NULL,
            memory_max REAL NOT NULL,
            disk_avg REAL NOT NULL,
            net_rx_total INTEGER NOT NULL,
            net_tx_total INTEGER NOT NULL,
            uptime_percent REAL NOT NULL,
            ping_avg REAL,
            sample_count INTEGER NOT NULL,
            UNIQUE(server_id, date)
        );
        
        -- Create indexes
        CREATE INDEX IF NOT EXISTS idx_metrics_raw_server_time ON metrics_raw(server_id, timestamp);
        CREATE INDEX IF NOT EXISTS idx_metrics_hourly_server_time ON metrics_hourly(server_id, hour_start);
        CREATE INDEX IF NOT EXISTS idx_metrics_daily_server_time ON metrics_daily(server_id, date);
    "#)?;
    
    // Add ping_ms column if it doesn't exist (migration for existing databases)
    let _ = conn.execute("ALTER TABLE metrics_raw ADD COLUMN ping_ms REAL", []);
    let _ = conn.execute("ALTER TABLE metrics_hourly ADD COLUMN ping_avg REAL", []);
    let _ = conn.execute("ALTER TABLE metrics_daily ADD COLUMN ping_avg REAL", []);
    
    Ok(conn)
}

pub fn store_metrics(conn: &Connection, server_id: &str, metrics: &SystemMetrics) -> rusqlite::Result<()> {
    let disk_usage = metrics.disks.first().map(|d| d.usage_percent).unwrap_or(0.0);
    
    // Get average ping latency from all targets
    let ping_ms: Option<f64> = metrics.ping.as_ref().and_then(|p| {
        let valid_pings: Vec<f64> = p.targets.iter()
            .filter_map(|t| t.latency_ms)
            .collect();
        if valid_pings.is_empty() {
            None
        } else {
            Some(valid_pings.iter().sum::<f64>() / valid_pings.len() as f64)
        }
    });
    
    conn.execute(
        r#"INSERT INTO metrics_raw (server_id, timestamp, cpu_usage, memory_usage, disk_usage, net_rx, net_tx, load_1, load_5, load_15, ping_ms)
           VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)"#,
        params![
            server_id,
            metrics.timestamp.to_rfc3339(),
            metrics.cpu.usage,
            metrics.memory.usage_percent,
            disk_usage,
            metrics.network.total_rx as i64,
            metrics.network.total_tx as i64,
            metrics.load_average.one,
            metrics.load_average.five,
            metrics.load_average.fifteen,
            ping_ms,
        ],
    )?;
    Ok(())
}

pub fn aggregate_hourly(conn: &Connection) -> rusqlite::Result<()> {
    let hour_ago = Utc::now() - Duration::hours(1);
    let hour_start = hour_ago.format("%Y-%m-%dT%H:00:00Z").to_string();
    
    conn.execute(
        r#"INSERT OR REPLACE INTO metrics_hourly (server_id, hour_start, cpu_avg, cpu_max, memory_avg, memory_max, disk_avg, net_rx_total, net_tx_total, sample_count)
           SELECT 
               server_id,
               strftime('%Y-%m-%dT%H:00:00Z', timestamp) as hour,
               AVG(cpu_usage),
               MAX(cpu_usage),
               AVG(memory_usage),
               MAX(memory_usage),
               AVG(disk_usage),
               MAX(net_rx) - MIN(net_rx),
               MAX(net_tx) - MIN(net_tx),
               COUNT(*)
           FROM metrics_raw
           WHERE timestamp >= ?1 AND timestamp < datetime(?1, '+1 hour')
           GROUP BY server_id, hour"#,
        params![hour_start],
    )?;
    Ok(())
}

pub fn aggregate_daily(conn: &Connection) -> rusqlite::Result<()> {
    let yesterday = (Utc::now() - Duration::days(1)).format("%Y-%m-%d").to_string();
    
    conn.execute(
        r#"INSERT OR REPLACE INTO metrics_daily (server_id, date, cpu_avg, cpu_max, memory_avg, memory_max, disk_avg, net_rx_total, net_tx_total, uptime_percent, sample_count)
           SELECT 
               server_id,
               date(hour_start) as day,
               AVG(cpu_avg),
               MAX(cpu_max),
               AVG(memory_avg),
               MAX(memory_max),
               AVG(disk_avg),
               SUM(net_rx_total),
               SUM(net_tx_total),
               (COUNT(*) * 100.0 / 24.0),
               SUM(sample_count)
           FROM metrics_hourly
           WHERE date(hour_start) = ?1
           GROUP BY server_id, day"#,
        params![yesterday],
    )?;
    Ok(())
}

pub fn cleanup_old_data(conn: &Connection) -> rusqlite::Result<()> {
    // Delete raw data older than 24 hours
    let cutoff_raw = (Utc::now() - Duration::hours(24)).to_rfc3339();
    conn.execute("DELETE FROM metrics_raw WHERE timestamp < ?1", params![cutoff_raw])?;
    
    // Delete hourly data older than 30 days
    let cutoff_hourly = (Utc::now() - Duration::days(30)).to_rfc3339();
    conn.execute("DELETE FROM metrics_hourly WHERE hour_start < ?1", params![cutoff_hourly])?;
    
    Ok(())
}

