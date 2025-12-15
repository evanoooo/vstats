-- VStats Cloud Database Schema
-- PostgreSQL 15+

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ========================================
-- 1. Users Table - Core user information
-- ========================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE,
    email_verified BOOLEAN DEFAULT FALSE,
    avatar_url TEXT,
    plan VARCHAR(20) DEFAULT 'free' CHECK (plan IN ('free', 'pro', 'enterprise')),
    server_limit INTEGER DEFAULT 5,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);

-- ========================================
-- 2. OAuth Providers - OAuth login info
-- ========================================
CREATE TABLE oauth_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('github', 'google')),
    provider_user_id VARCHAR(255) NOT NULL,
    provider_username VARCHAR(255),
    provider_email VARCHAR(255),
    provider_avatar_url TEXT,
    access_token TEXT,  -- Encrypted in application layer
    refresh_token TEXT, -- Encrypted in application layer
    token_expires_at TIMESTAMPTZ,
    raw_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_providers_user_id ON oauth_providers(user_id);
CREATE INDEX idx_oauth_providers_provider ON oauth_providers(provider, provider_user_id);

-- ========================================
-- 3. Sessions - User login sessions
-- ========================================
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(128) NOT NULL UNIQUE, -- SHA-256 hash of session token
    ip_address INET,
    user_agent TEXT,
    device_info JSONB DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    last_activity_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- ========================================
-- 4. Servers - Monitored server info
-- ========================================
CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255),
    ip_address INET,
    agent_key VARCHAR(64) UNIQUE NOT NULL, -- Unique key for agent authentication
    agent_version VARCHAR(50),
    os_type VARCHAR(50),
    os_version VARCHAR(100),
    status VARCHAR(20) DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'warning', 'error')),
    last_seen_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_servers_user_id ON servers(user_id);
CREATE INDEX idx_servers_agent_key ON servers(agent_key);
CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_last_seen_at ON servers(last_seen_at);

-- ========================================
-- 5. Server Metrics - Time series data
-- ========================================
CREATE TABLE server_metrics (
    id BIGSERIAL PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- CPU metrics
    cpu_usage DECIMAL(5,2),
    cpu_cores INTEGER,
    load_avg_1 DECIMAL(6,2),
    load_avg_5 DECIMAL(6,2),
    load_avg_15 DECIMAL(6,2),
    -- Memory metrics
    memory_total BIGINT,
    memory_used BIGINT,
    memory_free BIGINT,
    memory_cached BIGINT,
    memory_buffers BIGINT,
    swap_total BIGINT,
    swap_used BIGINT,
    -- Disk metrics (aggregated)
    disk_total BIGINT,
    disk_used BIGINT,
    disk_free BIGINT,
    -- Network metrics (aggregated)
    network_rx_bytes BIGINT,
    network_tx_bytes BIGINT,
    network_rx_packets BIGINT,
    network_tx_packets BIGINT,
    -- Process info
    process_count INTEGER,
    -- Raw data for detailed metrics
    raw_data JSONB DEFAULT '{}'
);

-- Partition by time for better performance (optional, enable if needed)
-- CREATE INDEX idx_server_metrics_server_id_time ON server_metrics(server_id, collected_at DESC);
CREATE INDEX idx_server_metrics_server_id ON server_metrics(server_id);
CREATE INDEX idx_server_metrics_collected_at ON server_metrics(collected_at DESC);

-- ========================================
-- 6. Alerts - Alert rules and history
-- ========================================
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE, -- NULL = all servers
    name VARCHAR(255) NOT NULL,
    description TEXT,
    metric_type VARCHAR(50) NOT NULL CHECK (metric_type IN ('cpu', 'memory', 'disk', 'network', 'status')),
    condition VARCHAR(20) NOT NULL CHECK (condition IN ('gt', 'gte', 'lt', 'lte', 'eq', 'neq')),
    threshold DECIMAL(10,2) NOT NULL,
    duration_seconds INTEGER DEFAULT 60, -- How long condition must persist
    notification_channels JSONB DEFAULT '[]', -- ['email', 'webhook', 'slack']
    is_enabled BOOLEAN DEFAULT TRUE,
    cooldown_seconds INTEGER DEFAULT 300, -- Minimum time between alerts
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_alert_rules_user_id ON alert_rules(user_id);
CREATE INDEX idx_alert_rules_server_id ON alert_rules(server_id);

CREATE TABLE alert_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('triggered', 'resolved', 'acknowledged')),
    triggered_value DECIMAL(10,2),
    triggered_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES users(id),
    notification_sent BOOLEAN DEFAULT FALSE,
    notes TEXT
);

CREATE INDEX idx_alert_history_rule_id ON alert_history(rule_id);
CREATE INDEX idx_alert_history_server_id ON alert_history(server_id);
CREATE INDEX idx_alert_history_triggered_at ON alert_history(triggered_at DESC);

-- ========================================
-- 7. API Keys - For external integrations
-- ========================================
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(8) NOT NULL, -- First 8 chars for identification
    key_hash VARCHAR(128) NOT NULL UNIQUE, -- SHA-256 hash of full key
    permissions JSONB DEFAULT '["read"]', -- ['read', 'write', 'admin']
    rate_limit INTEGER DEFAULT 1000, -- Requests per hour
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- ========================================
-- 8. Audit Logs - Security audit trail
-- ========================================
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    request_data JSONB DEFAULT '{}',
    response_status INTEGER,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- ========================================
-- 9. Auth Reports - OAuth Authorization Reports from Sites
-- ========================================
CREATE TABLE auth_reports (
    id BIGSERIAL PRIMARY KEY,
    site_url VARCHAR(512) NOT NULL,
    site_host VARCHAR(255) NOT NULL,  -- Extracted hostname for grouping
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('github', 'google')),
    username VARCHAR(255) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    reported_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_auth_reports_site_host ON auth_reports(site_host);
CREATE INDEX idx_auth_reports_username ON auth_reports(username);
CREATE INDEX idx_auth_reports_provider ON auth_reports(provider);
CREATE INDEX idx_auth_reports_reported_at ON auth_reports(reported_at DESC);
CREATE INDEX idx_auth_reports_date ON auth_reports((reported_at::DATE));

-- View for daily auth statistics
CREATE OR REPLACE VIEW auth_daily_stats AS
SELECT 
    reported_at::DATE AS date,
    COUNT(DISTINCT site_host) AS unique_sites,
    COUNT(DISTINCT username) AS unique_users,
    COUNT(*) AS total_auths,
    COUNT(DISTINCT CASE WHEN provider = 'github' THEN username END) AS github_users,
    COUNT(DISTINCT CASE WHEN provider = 'google' THEN username END) AS google_users
FROM auth_reports
GROUP BY reported_at::DATE
ORDER BY date DESC;

-- View for site statistics
CREATE OR REPLACE VIEW auth_site_stats AS
SELECT 
    site_host,
    site_url,
    COUNT(DISTINCT username) AS unique_users,
    COUNT(*) AS total_auths,
    MIN(reported_at) AS first_seen,
    MAX(reported_at) AS last_seen,
    COUNT(DISTINCT reported_at::DATE) AS active_days
FROM auth_reports
GROUP BY site_host, site_url
ORDER BY last_seen DESC;

-- ========================================
-- 10. Subscription & Billing (Optional)
-- ========================================
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan VARCHAR(20) NOT NULL CHECK (plan IN ('free', 'pro', 'enterprise')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'past_due', 'expired')),
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_stripe_customer_id ON subscriptions(stripe_customer_id);

-- ========================================
-- Functions & Triggers
-- ========================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to relevant tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_oauth_providers_updated_at BEFORE UPDATE ON oauth_providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old metrics (keep last 30 days by default)
CREATE OR REPLACE FUNCTION cleanup_old_metrics(days_to_keep INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM server_metrics 
    WHERE collected_at < CURRENT_TIMESTAMP - (days_to_keep || ' days')::INTERVAL;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ========================================
-- Views for common queries
-- ========================================

-- User dashboard summary view
CREATE OR REPLACE VIEW user_dashboard_summary AS
SELECT 
    u.id AS user_id,
    u.username,
    u.email,
    u.plan,
    u.server_limit,
    COUNT(DISTINCT s.id) FILTER (WHERE s.deleted_at IS NULL) AS server_count,
    COUNT(DISTINCT s.id) FILTER (WHERE s.status = 'online' AND s.deleted_at IS NULL) AS online_servers,
    COUNT(DISTINCT ah.id) FILTER (WHERE ah.status = 'triggered' AND ah.triggered_at > CURRENT_TIMESTAMP - INTERVAL '24 hours') AS alerts_24h
FROM users u
LEFT JOIN servers s ON u.id = s.user_id
LEFT JOIN alert_history ah ON s.id = ah.server_id
WHERE u.status = 'active' AND u.deleted_at IS NULL
GROUP BY u.id, u.username, u.email, u.plan, u.server_limit;

-- Latest server metrics view
CREATE OR REPLACE VIEW latest_server_metrics AS
SELECT DISTINCT ON (server_id)
    sm.*,
    s.name AS server_name,
    s.hostname,
    s.status AS server_status
FROM server_metrics sm
JOIN servers s ON sm.server_id = s.id
ORDER BY server_id, collected_at DESC;

-- ========================================
-- Initial Data
-- ========================================

-- Create default system user for background tasks
INSERT INTO users (id, username, email, plan, status, metadata)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    'system',
    'system@vstats.local',
    'enterprise',
    'active',
    '{"is_system": true}'
) ON CONFLICT DO NOTHING;

COMMENT ON TABLE users IS 'Core user accounts';
COMMENT ON TABLE oauth_providers IS 'OAuth provider connections (GitHub, Google)';
COMMENT ON TABLE sessions IS 'Active user sessions';
COMMENT ON TABLE servers IS 'Monitored server registrations';
COMMENT ON TABLE server_metrics IS 'Time series metrics data';
COMMENT ON TABLE alert_rules IS 'User-defined alerting rules';
COMMENT ON TABLE alert_history IS 'Alert trigger history';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access';
COMMENT ON TABLE audit_logs IS 'Security audit trail';
COMMENT ON TABLE subscriptions IS 'Subscription and billing information';
