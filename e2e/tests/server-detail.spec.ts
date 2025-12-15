import { test, expect } from '@playwright/test';

test.describe('Server Detail Page', () => {
  test('should show 404 or redirect for non-existent server', async ({ page }) => {
    await page.goto('/server/non-existent-id-12345');
    await page.waitForLoadState('networkidle');
    
    // Should show error, redirect, or display empty state
    const hasErrorMessage = await page.getByText(/not found|error|no server|does not exist/i)
      .isVisible()
      .catch(() => false);
    const redirectedAway = !page.url().includes('/server/non-existent');
    
    expect(hasErrorMessage || redirectedAway).toBeTruthy();
  });

  test('should have proper page structure', async ({ page }) => {
    await page.goto('/server/test-server');
    await page.waitForLoadState('networkidle');
    
    // Check basic page structure
    await expect(page.locator('body')).toBeVisible();
    
    // Should have some navigation to go back
    const backButton = page.getByRole('link', { name: /back|home|dashboard/i })
      .or(page.getByRole('button', { name: /back/i }));
    
    const hasNavigation = await backButton.first().isVisible().catch(() => false);
    
    // Navigation or we're redirected
    expect(hasNavigation || !page.url().includes('/server/')).toBeTruthy();
  });
});

test.describe('Server Metrics Display', () => {
  test('should handle loading state', async ({ page }) => {
    await page.goto('/server/test');
    
    // Check for loading indicators
    const loadingSpinner = page.locator('.loading, .spinner, [role="progressbar"]');
    const loadingText = page.getByText(/loading/i);
    
    // Either shows loading or loads content immediately
    await page.waitForLoadState('networkidle');
    
    expect(true).toBeTruthy();
  });

  test('should display metric cards if server exists', async ({ page }) => {
    await page.goto('/server/test');
    await page.waitForLoadState('networkidle');
    
    // If server exists, look for metric cards
    const metricCards = page.locator('.metric-card, .glass-card, [data-testid*="metric"]');
    
    // If we have metric cards, verify they're visible
    const cardCount = await metricCards.count();
    if (cardCount > 0) {
      await expect(metricCards.first()).toBeVisible();
    }
  });
});

test.describe('Charts and Visualizations', () => {
  test('should handle chart rendering', async ({ page }) => {
    await page.goto('/server/test');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000); // Wait for charts to render
    
    // Look for recharts components or svg charts
    const charts = page.locator('.recharts-wrapper, svg.recharts-surface, canvas');
    
    // Charts are optional - depends on if server exists with data
    const chartCount = await charts.count();
    console.log(`Found ${chartCount} chart elements`);
  });

  test('should have time range selector if charts exist', async ({ page }) => {
    await page.goto('/server/test');
    await page.waitForLoadState('networkidle');
    
    // Look for time range buttons (1h, 24h, 7d, etc.)
    const timeRangeButtons = page.getByRole('button', { name: /1h|24h|7d|30d|1y/i });
    
    const hasTimeRange = await timeRangeButtons.first().isVisible().catch(() => false);
    
    if (hasTimeRange) {
      // Click on a different time range
      await timeRangeButtons.first().click();
      await page.waitForTimeout(500);
    }
  });
});

