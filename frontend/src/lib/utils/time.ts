export function timeAgo(dateStr: string | null): string {
	if (!dateStr) return '';
	const date = new Date(dateStr);
	if (isNaN(date.getTime())) return '';
	const now = new Date();
	const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

	// Future dates (e.g. clock skew) read as "just now" rather than negative/garbage.
	if (seconds < 60) return 'just now';
	if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
	if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
	if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;

	return date.toLocaleDateString();
}
