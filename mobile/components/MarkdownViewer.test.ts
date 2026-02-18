import { clampMarkdownContent, MAX_MARKDOWN_CONTENT_CHARS } from './MarkdownViewer';

describe('MarkdownViewer guardrails', () => {
  it('keeps small markdown content unchanged', () => {
    const content = '# Hello\n\nShort content';
    expect(clampMarkdownContent(content)).toBe(content);
  });

  it('truncates oversized markdown content with a clear marker', () => {
    const oversized = 'a'.repeat(MAX_MARKDOWN_CONTENT_CHARS + 2500);

    const clamped = clampMarkdownContent(oversized);

    expect(clamped.length).toBeGreaterThan(MAX_MARKDOWN_CONTENT_CHARS);
    expect(clamped).toContain('_Output truncated (2500 characters omitted)_');
    expect(clamped.startsWith('a'.repeat(MAX_MARKDOWN_CONTENT_CHARS))).toBe(true);
  });
});
