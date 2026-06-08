// highlightParts splits `text` into runs, marking the runs that match any
// whitespace-separated token in `term`. This powers search-result highlighting
// without ever rendering untrusted HTML: callers wrap the `hit` runs in <mark>
// using normal Svelte text interpolation, so there is no {@html}/XSS surface.
export function highlightParts(text: string, term: string): { text: string; hit: boolean }[] {
	if (!term || !text) return [{ text, hit: false }];
	const tokens = term
		.trim()
		.split(/\s+/)
		.filter(Boolean)
		.map(escapeRegExp);
	if (tokens.length === 0) return [{ text, hit: false }];

	const re = new RegExp(`(${tokens.join('|')})`, 'gi');
	const parts: { text: string; hit: boolean }[] = [];
	let last = 0;
	let m: RegExpExecArray | null;
	while ((m = re.exec(text)) !== null) {
		if (m.index > last) parts.push({ text: text.slice(last, m.index), hit: false });
		parts.push({ text: m[0], hit: true });
		last = m.index + m[0].length;
		if (m.index === re.lastIndex) re.lastIndex++; // guard against zero-width matches
	}
	if (last < text.length) parts.push({ text: text.slice(last), hit: false });
	return parts;
}

function escapeRegExp(s: string): string {
	return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
