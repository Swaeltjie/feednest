import { describe, it, expect, vi, afterEach } from 'vitest';
import { timeAgo } from './time';

describe('timeAgo', () => {
	afterEach(() => {
		vi.useRealTimers();
	});

	it('returns empty string for null', () => {
		expect(timeAgo(null)).toBe('');
	});

	it('returns "just now" for <60 seconds ago', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-01-15T12:00:30Z'));
		expect(timeAgo('2026-01-15T12:00:00Z')).toBe('just now');
	});

	it('returns minutes ago', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-01-15T12:05:00Z'));
		expect(timeAgo('2026-01-15T12:00:00Z')).toBe('5m ago');
	});

	it('returns hours ago', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-01-15T15:00:00Z'));
		expect(timeAgo('2026-01-15T12:00:00Z')).toBe('3h ago');
	});

	it('returns days ago', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-01-18T12:00:00Z'));
		expect(timeAgo('2026-01-15T12:00:00Z')).toBe('3d ago');
	});

	it('returns formatted date for >7 days', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-02-15T12:00:00Z'));
		const result = timeAgo('2026-01-15T12:00:00Z');
		expect(result).not.toContain('ago');
		expect(result).not.toBe('');
	});
});
