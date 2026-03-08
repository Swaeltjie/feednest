import DOMPurify from 'isomorphic-dompurify';
import { isSafeUrl } from '$lib/api/client';

let initialized = false;

export function initSanitizer() {
	if (initialized) return;
	initialized = true;

	DOMPurify.addHook('afterSanitizeAttributes', (node: Element) => {
		if (node.tagName === 'A') {
			const href = node.getAttribute('href');
			if (href && !isSafeUrl(href)) {
				node.removeAttribute('href');
			}
			if (node.getAttribute('target') === '_blank') {
				node.setAttribute('rel', 'noopener noreferrer');
			}
		}
		node.removeAttribute('style');
	});
}

export { DOMPurify };
