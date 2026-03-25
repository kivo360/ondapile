// tests/evals/phase5.spec.ts (Playwright)
// NOTE: These tests require Playwright and a running dev server.
// Run with: bunx playwright test tests/evals/phase5.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Phase 5: Frontend Integration', () => {
  test('P5.1: Signup works end-to-end', async ({ page }) => {
    await page.goto('/auth/signup');
    await page.getByLabel(/name/i).fill(`P5 User ${Date.now()}`);
    await page.getByLabel(/email/i).fill(`p5-${Date.now()}@test.com`);
    await page.getByLabel(/password/i).fill('P5Test1234!');
    const terms = page.getByRole('checkbox');
    if (await terms.isVisible()) await terms.check();
    await page.getByRole('button', { name: /sign up|create/i }).click();
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 });
  });

  test('P5.2: Dashboard loads without port 8080', async ({ page }) => {
    await page.goto('/dashboard/accounts');
    await expect(page.getByText(/unauthorized|ECONNREFUSED|8080/i)).not.toBeVisible();
    await expect(page.getByText(/accounts/i)).toBeVisible({ timeout: 10_000 });
  });

  test('P5.3: API keys page works', async ({ page }) => {
    await page.goto('/dashboard/api-keys');
    await expect(page.getByText(/api.?key/i)).toBeVisible({ timeout: 10_000 });
  });

  test('P5.4: No console errors referencing port 8080', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
    page.on('requestfailed', req => errors.push(req.url()));

    await page.goto('/dashboard');
    await page.waitForTimeout(3000);

    const port8080Errors = errors.filter(e => e.includes('8080'));
    expect(port8080Errors).toHaveLength(0);
  });

  test('P5.5: All sidebar navigation works', async ({ page }) => {
    await page.goto('/dashboard');
    const pages = ['/dashboard/accounts', '/dashboard/api-keys',
      '/dashboard/webhooks', '/dashboard/logs', '/dashboard/settings'];
    for (const p of pages) {
      await page.goto(p);
      await expect(page).toHaveURL(p);
      await expect(page.getByText(/error|500|404/i)).not.toBeVisible();
    }
  });
});
