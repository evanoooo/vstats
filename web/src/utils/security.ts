import type { SiteSettings, SocialLink } from '../types';

const allowedProtocols = new Set(['http:', 'https:', 'mailto:', 'tel:']);
const fallbackOrigin = typeof window !== 'undefined' && window.location?.origin
  ? window.location.origin
  : 'http://localhost';

// Basic URL sanitization to block javascript/data schemes while allowing http/https,
// mailto/tel, and same-origin relative paths.
export function sanitizeUrl(rawUrl?: string | null): string | null {
  if (!rawUrl) return null;
  const trimmed = rawUrl.trim();
  if (!trimmed) return null;

  try {
    const parsed = new URL(trimmed, fallbackOrigin);
    if (allowedProtocols.has(parsed.protocol)) {
      return trimmed;
    }
  } catch {
    // Ignore parsing errors and treat as unsafe
  }

  return null;
}

export function sanitizeSocialLinks(links: SocialLink[] | undefined | null): SocialLink[] {
  if (!links || links.length === 0) return [];

  return links
    .map(link => {
      const safeUrl = sanitizeUrl(link.url);
      if (!safeUrl) return null;
      return { ...link, url: safeUrl };
    })
    .filter(Boolean) as SocialLink[];
}

export function sanitizeSiteSettings(settings: SiteSettings | null | undefined): SiteSettings {
  if (!settings) {
    return { site_name: '', site_description: '', social_links: [] };
  }

  const sanitizedLinks = sanitizeSocialLinks(settings.social_links);

  let sanitizedTheme = settings.theme;
  if (settings.theme?.background?.custom_url) {
    const safeCustomUrl = sanitizeUrl(settings.theme.background.custom_url);
    sanitizedTheme = {
      ...settings.theme,
      background: {
        ...settings.theme.background,
        custom_url: safeCustomUrl || undefined
      }
    };
  }

  return {
    site_name: settings.site_name || '',
    site_description: settings.site_description || '',
    social_links: sanitizedLinks,
    theme: sanitizedTheme
  };
}
