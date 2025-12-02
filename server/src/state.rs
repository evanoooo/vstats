use axum::extract::ws::Message;
use rusqlite::Connection;
use std::{collections::HashMap, sync::Arc};
use tokio::sync::{broadcast, mpsc, Mutex, RwLock};

use crate::config::AppConfig;
use crate::types::AgentMetricsData;

/// Represents a connected agent's command channel
pub type AgentCommandSender = mpsc::Sender<Message>;

#[derive(Clone)]
pub struct AppState {
    pub config: Arc<RwLock<AppConfig>>,
    pub metrics_tx: broadcast::Sender<String>,
    pub agent_metrics: Arc<RwLock<HashMap<String, AgentMetricsData>>>,
    pub db: Arc<Mutex<Connection>>,
    /// Track connected agents by server_id -> command sender
    pub agent_connections: Arc<RwLock<HashMap<String, AgentCommandSender>>>,
}

