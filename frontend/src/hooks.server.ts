import type { Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const BACKEND_URL = env.BACKEND_URL || 'http://localhost:8082';

export const handle: Handle = async ({ event, resolve }) => {
	// Proxy /api/* requests to the backend
	if (event.url.pathname.startsWith('/api/') && !event.url.pathname.startsWith('/api/docs')) {
		const target = `${BACKEND_URL}${event.url.pathname}${event.url.search}`;
		const headers = new Headers(event.request.headers);
		headers.delete('host');

		const proxyResponse = await fetch(target, {
			method: event.request.method,
			headers,
			body: event.request.method !== 'GET' && event.request.method !== 'HEAD'
				? event.request.body
				: undefined,
			// @ts-expect-error duplex is needed for streaming request bodies
			duplex: 'half',
		});

		return new Response(proxyResponse.body, {
			status: proxyResponse.status,
			statusText: proxyResponse.statusText,
			headers: proxyResponse.headers,
		});
	}

	const response = await resolve(event);

	response.headers.set('X-Frame-Options', 'DENY');
	response.headers.set('X-Content-Type-Options', 'nosniff');
	response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
	response.headers.set('Permissions-Policy', 'geolocation=(), microphone=(), camera=()');
	response.headers.set(
		'Content-Security-Policy',
		`default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' https: data:; connect-src 'self'; font-src 'self'; frame-src 'none'; object-src 'none'; base-uri 'self'; form-action 'self'`
	);

	return response;
};
