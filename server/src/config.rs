use serde::{Deserialize, Serialize};
use std::{fs, path::PathBuf};

pub const JWT_SECRET: &str = "vstats-super-secret-key-change-in-production";
pub const CONFIG_FILE: &str = "vstats-config.json";
pub const DB_FILE: &str = "vstats.db";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppConfig {
    pub admin_password_hash: String,
    pub servers: Vec<RemoteServer>,
    #[serde(default)]
    pub site_settings: SiteSettings,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SiteSettings {
    #[serde(default)]
    pub site_name: String,
    #[serde(default)]
    pub site_description: String,
    #[serde(default)]
    pub social_links: Vec<SocialLink>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialLink {
    pub platform: String,
    pub url: String,
    #[serde(default)]
    pub label: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoteServer {
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub url: String,
    pub location: String,
    pub provider: String,
    #[serde(default)]
    pub token: String,
}

impl Default for AppConfig {
    fn default() -> Self {
        let hash = bcrypt::hash("admin", bcrypt::DEFAULT_COST).unwrap();
        Self {
            admin_password_hash: hash,
            servers: vec![],
            site_settings: SiteSettings {
                site_name: "xProb Dashboard".to_string(),
                site_description: "Real-time Server Monitoring".to_string(),
                social_links: vec![],
            },
        }
    }
}

pub fn load_config() -> AppConfig {
    let path = PathBuf::from(CONFIG_FILE);
    if path.exists() {
        let content = fs::read_to_string(&path).unwrap_or_default();
        serde_json::from_str(&content).unwrap_or_default()
    } else {
        let config = AppConfig::default();
        save_config(&config);
        config
    }
}

pub fn save_config(config: &AppConfig) {
    let content = serde_json::to_string_pretty(config).unwrap();
    fs::write(CONFIG_FILE, content).ok();
}

