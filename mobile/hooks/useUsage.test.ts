import { getUsageRefetchInterval } from './useUsage';

describe('useUsage polling', () => {
  it('polls only when app state is active', () => {
    expect(getUsageRefetchInterval('active')).toBe(30000);
    expect(getUsageRefetchInterval('inactive')).toBe(false);
    expect(getUsageRefetchInterval('background')).toBe(false);
  });
});
