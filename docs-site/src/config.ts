// OAuth Proxy configuration
// The OAuth proxy is deployed on Cloudflare Workers
export const OAUTH_PROXY_URL = 'https://vstats-oauth-proxy.zsai001.workers.dev';

// Admin email address - has ability to send broadcast emails
export const ADMIN_EMAIL = 'tsiannian@gmail.com';

// Check if user is admin
export function isAdmin(email: string | null | undefined): boolean {
    if (!email) return false;
    return email.toLowerCase() === ADMIN_EMAIL.toLowerCase();
}
