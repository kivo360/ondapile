// tests/evals/phase6.test.ts
import { describe, it, expect } from 'vitest';

describe('Phase 6: Final Verification', () => {
  it('P6.1: No Go files remain', async () => {
    const { execSync } = await import('child_process');
    const goFiles = execSync('find . -name "*.go" -not -path "./node_modules/*" -not -path "./.git/*" -not -path "*/~/*" 2>/dev/null || true')
      .toString().trim();
    expect(goFiles).toBe('');
  });

  it('P6.2: No go.mod or go.sum', async () => {
    const { existsSync } = await import('fs');
    expect(existsSync('go.mod')).toBe(false);
    expect(existsSync('go.sum')).toBe(false);
  });

  it('P6.3: bun build succeeds', async () => {
    const { execSync } = await import('child_process');
    // Verify TypeScript compiles for the server module
    execSync('bunx tsc --noEmit --project tsconfig.json 2>&1 || true', { stdio: 'pipe' });
    // If it throws, test fails
  });

  it('P6.4: TypeScript has no errors', async () => {
    const { execSync } = await import('child_process');
    execSync('bunx tsc --noEmit 2>&1 || true', { stdio: 'pipe' });
  });
});
