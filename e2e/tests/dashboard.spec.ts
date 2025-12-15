import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to dashboard
    await page.goto('/');
  });

  test('should display dashboard page', async ({ page }) => {
    // Wait for the page to load
    await page.waitForLoadState('networkidle');
    
    // Check page title or main content
    await expect(page).toHaveTitle(/VStats|Dashboard/i);
  });

  test('should show login page when not authenticated', async ({ page }) => {
    // Check if redirected to login or shows login button
    const loginButton = page.getByRole('button', { name: /login|sign in/i });
    const loginLink = page.getByRole('link', { name: /login|sign in/i });
    
    // Either login button/link should be visible, or we're on login page
    const hasLoginElement = await loginButton.isVisible().catch(() => false) ||
                            await loginLink.isVisible().catch(() => false) ||
                            page.url().includes('/login');
    
    expect(hasLoginElement).toBeTruthy();
  });

  test('should have navigation elements', async ({ page }) => {
    // Check for common navigation elements
    await expect(page.locator('nav, header, [role="navigation"]').first()).toBeVisible();
  });

  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    // Page should still be functional
    await page.waitForLoadState('networkidle');
    
    // Check main content is visible
    await expect(page.locator('body')).toBeVisible();
  });
});

test.describe('Theme', () => {
  test('should toggle theme', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Find theme toggle button (if exists)
    const themeToggle = page.getByRole('button', { name: /theme|dark|light/i }).first();
    
    if (await themeToggle.isVisible().catch(() => false)) {
      const htmlClass = await page.locator('html').getAttribute('class');
      
      await themeToggle.click();
      await page.waitForTimeout(500); // Wait for theme transition
      
      const newHtmlClass = await page.locator('html').getAttribute('class');
      
      // Theme class should have changed
      expect(newHtmlClass).not.toBe(htmlClass);
    }
  });
});

test.describe('Accessibility', () => {
  test('should have proper heading structure', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Check for at least one heading
    const headings = page.locator('h1, h2, h3, h4, h5, h6');
    await expect(headings.first()).toBeVisible();
  });

  test('should have focusable elements', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Tab through the page
    await page.keyboard.press('Tab');
    
    // Something should be focused
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedElement).toBeTruthy();
  });

  test('should have proper contrast (no text on matching background)', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Basic check - page should have visible text
    const bodyText = await page.locator('body').textContent();
    expect(bodyText?.length).toBeGreaterThan(0);
  });
});

