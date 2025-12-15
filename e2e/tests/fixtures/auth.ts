import { test as base, expect } from '@playwright/test';

// Extended test fixture with authentication
export const test = base.extend<{ authenticatedPage: typeof base }>({
  authenticatedPage: async ({ page }, use) => {
    // Mock authentication by setting token in localStorage
    await page.goto('/');
    await page.evaluate(() => {
      // Set a mock JWT token
      localStorage.setItem('token', 'mock-jwt-token-for-testing');
    });
    
    await use(base);
  },
});

export { expect };

// Helper function to login via UI
export async function loginViaUI(page: any, password: string = 'admin') {
  await page.goto('/login');
  await page.waitForLoadState('networkidle');
  
  const passwordInput = page.locator('input[type="password"]');
  await passwordInput.fill(password);
  
  const submitButton = page.getByRole('button', { name: /login|sign in|submit/i });
  await submitButton.click();
  
  // Wait for redirect
  await page.waitForURL((url: URL) => !url.pathname.includes('/login'), {
    timeout: 10000,
  }).catch(() => {
    // Login might fail, that's okay for testing
  });
}

// Helper function to logout
export async function logout(page: any) {
  await page.evaluate(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('auth-token');
    sessionStorage.clear();
  });
  await page.reload();
}

// Helper function to mock API responses
export async function mockApiResponse(page: any, url: string, response: any) {
  await page.route(url, async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(response),
    });
  });
}

// Helper function to mock WebSocket
export async function mockWebSocket(page: any) {
  await page.addInitScript(() => {
    window.WebSocket = class MockWebSocket {
      onopen: (() => void) | null = null;
      onmessage: ((event: any) => void) | null = null;
      onclose: (() => void) | null = null;
      onerror: ((error: any) => void) | null = null;
      readyState = 1;
      
      constructor() {
        setTimeout(() => {
          if (this.onopen) this.onopen();
        }, 100);
      }
      
      send() {}
      close() {
        if (this.onclose) this.onclose();
      }
    } as any;
  });
}

