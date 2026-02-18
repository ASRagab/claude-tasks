import { cronToHuman } from './cronToHuman';

describe('cronToHuman guardrails', () => {
  it('keeps valid schedules human-readable', () => {
    expect(cronToHuman('0 */5 * * * *')).toBe('Every 5 minutes');
  });

  it('falls back to raw expression for oversized ranges', () => {
    const oversized = '0 0-1000000 * * * *';
    expect(cronToHuman(oversized)).toBe(oversized);
  });

  it('falls back to raw expression for out-of-bounds values', () => {
    const invalid = '0 99 * * * *';
    expect(cronToHuman(invalid)).toBe(invalid);
  });

  it('returns raw expression when cron input is excessively long', () => {
    const tooLong = `${'0 '.repeat(600)}`.trim();
    expect(cronToHuman(tooLong)).toBe(tooLong);
  });
});
