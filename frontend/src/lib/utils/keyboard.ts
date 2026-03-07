export interface KeyboardShortcuts {
	[key: string]: (e: KeyboardEvent) => void;
}

export function setupKeyboardShortcuts(shortcuts: KeyboardShortcuts) {
	let chordBuffer = '';
	let chordTimeout: ReturnType<typeof setTimeout> | null = null;

	const handler = (e: KeyboardEvent) => {
		const target = e.target as HTMLElement;
		const tag = target.tagName;
		const isInput = tag === 'INPUT' || tag === 'TEXTAREA' || target.isContentEditable;

		// Cmd+K / Ctrl+K always fires (even in inputs)
		if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
			e.preventDefault();
			shortcuts['cmd+k']?.(e);
			return;
		}

		// ? always fires for help overlay (unless in input)
		if (e.key === '?' && !isInput) {
			e.preventDefault();
			shortcuts['?']?.(e);
			return;
		}

		if (isInput) return;

		const key = e.key;

		// Handle chord completion (e.g., second 'g' in 'gg')
		if (chordBuffer) {
			const chord = chordBuffer + key.toLowerCase();
			if (chordTimeout) clearTimeout(chordTimeout);
			chordBuffer = '';
			if (shortcuts[chord]) {
				e.preventDefault();
				shortcuts[chord](e);
				return;
			}
		}

		// Check if this key starts a possible chord
		const lowerKey = key.toLowerCase();
		const possibleChords = Object.keys(shortcuts).filter(
			(k) => k.length === 2 && k.startsWith(lowerKey) && !k.includes('+')
		);

		if (possibleChords.length > 0 && !shortcuts[lowerKey]) {
			// This key only starts chords, no single-key binding
			chordBuffer = lowerKey;
			chordTimeout = setTimeout(() => {
				chordBuffer = '';
			}, 300);
			e.preventDefault();
			return;
		}

		if (possibleChords.length > 0 && shortcuts[lowerKey]) {
			// Has both chord and single-key binding - buffer for chord
			chordBuffer = lowerKey;
			chordTimeout = setTimeout(() => {
				if (shortcuts[chordBuffer]) {
					shortcuts[chordBuffer]({} as KeyboardEvent);
				}
				chordBuffer = '';
			}, 300);
			e.preventDefault();
			return;
		}

		// Check original case key (for Shift+key shortcuts like 'G')
		if (shortcuts[key]) {
			e.preventDefault();
			shortcuts[key](e);
			return;
		}

		// Simple single-key shortcut
		if (shortcuts[lowerKey]) {
			e.preventDefault();
			shortcuts[lowerKey](e);
		}
	};

	document.addEventListener('keydown', handler);
	return () => document.removeEventListener('keydown', handler);
}
