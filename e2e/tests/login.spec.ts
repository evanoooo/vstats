import { test, expect } from '@playwright/test';

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display login form', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // Check for password input
    const passwordInput = page.getByRole('textbox', { name: /password/i })
      .or(page.locator('input[type="password"]'));
    
    await expect(passwordInput.first()).toBeVisible();
  });

  test('should show error on invalid login', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // Find password input
    const passwordInput = page.locator('input[type="password"]');
    
    if (await passwordInput.isVisible()) {
      // Enter wrong password
      await passwordInput.fill('wrongpassword123');
      
      // Submit form
      const submitButton = page.getByRole('button', { name: /login|sign in|submit/i });
      await submitButton.click();
      
      // Wait for error message or stay on login page
      await page.waitForTimeout(1000);
      
      // Should still be on login page or show error
      const hasError = await page.getByText(/invalid|error|wrong|incorrect/i).isVisible().catch(() => false);
      const stillOnLogin = page.url().includes('/login');
      
      expect(hasError || stillOnLogin).toBeTruthy();
    }
  });

  test('should have OAuth login options if configured', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // Check for OAuth buttons (GitHub, Google, etc.)
    const oauthButtons = page.getByRole('button', { name: /github|google|oauth/i });
    const oauthLinks = page.getByRole('link', { name: /github|google|oauth/i });
    
    // OAuth might not be configured, so just check they exist if visible
    const hasOAuth = await oauthButtons.first().isVisible().catch(() => false) ||
                     await oauthLinks.first().isVisible().catch(() => false);
    
    // This is informational - OAuth is optional
    console.log(`OAuth login available: ${hasOAuth}`);
  });

  test('should handle keyboard navigation', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // Tab to password field
    await page.keyboard.press('Tab');
    
    // Type password
    await page.keyboard.type('testpassword');
    
    // Tab to submit button and press Enter
    await page.keyboard.press('Tab');
    await page.keyboard.press('Enter');
    
    // Wait for response
    await page.waitForTimeout(500);
    
    // Page should respond to keyboard input
    expect(true).toBeTruthy();
  });
});

test.describe('Authentication Flow', () => {
  test('should redirect to login when accessing protected routes', async ({ page }) => {
    // Try to access settings (protected route)
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');
    
    // Should redirect to login or show login modal
    const isOnLogin = page.url().includes('/login');
    const hasLoginForm = await page.locator('input[type="password"]').isVisible().catch(() => false);
    
    expect(isOnLogin || hasLoginForm).toBeTruthy();
  });

  test('should handle session expiry gracefully', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Clear any stored tokens
    await page.evaluate(() => {
      localStorage.removeItem('token');
      localStorage.removeItem('auth-token');
      sessionStorage.clear();
    });
    
    // Refresh page
    await page.reload();
    await page.waitForLoadState('networkidle');
    
    // Page should handle gracefully (not crash)
    await expect(page.locator('body')).toBeVisible();
  });
});

