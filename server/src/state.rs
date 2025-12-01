use rusqlite::Connection;
use std::{collections::HashMap, sync::Arc};
use tokio::sync::{broadcast, Mutex, RwLock};

use crate::config::AppConfig;
use crate::types::AgentMetricsData;

#[derive(Clone)]
pub struct AppState {
    pub config: Arc<RwLock<AppConfig>>,
    pub metrics_tx: broadcast::Sender<String>,
    pub agent_metrics: Arc<RwLock<HashMap<String, AgentMetricsData>>>,
    pub db: Arc<Mutex<Connection>>,
}

