const colorCache = new Map<string, string>();

export function hashToColor(str: string): string {
	let hash = 0;
	for (let i = 0; i < str.length; i++) {
		hash = str.charCodeAt(i) + ((hash << 5) - hash);
	}
	const hue = Math.abs(hash) % 360;
	return `hsl(${hue}, 65%, 55%)`;
}

export function extractDominantColor(imageUrl: string): Promise<string> {
	if (colorCache.has(imageUrl)) {
		return Promise.resolve(colorCache.get(imageUrl)!);
	}

	return new Promise((resolve) => {
		const img = new Image();
		img.crossOrigin = 'anonymous';

		img.onload = () => {
			try {
				const canvas = document.createElement('canvas');
				canvas.width = 16;
				canvas.height = 16;
				const ctx = canvas.getContext('2d');
				if (!ctx) { resolve(hashToColor(imageUrl)); return; }

				ctx.drawImage(img, 0, 0, 16, 16);
				const data = ctx.getImageData(0, 0, 16, 16).data;

				let r = 0, g = 0, b = 0, count = 0;
				for (let i = 0; i < data.length; i += 4) {
					const brightness = (data[i] + data[i + 1] + data[i + 2]) / 3;
					if (brightness > 30 && brightness < 225) {
						r += data[i];
						g += data[i + 1];
						b += data[i + 2];
						count++;
					}
				}

				if (count === 0) { resolve(hashToColor(imageUrl)); return; }

				const color = `rgb(${Math.round(r / count)}, ${Math.round(g / count)}, ${Math.round(b / count)})`;
				colorCache.set(imageUrl, color);
				resolve(color);
			} catch {
				resolve(hashToColor(imageUrl));
			}
		};

		img.onerror = () => resolve(hashToColor(imageUrl));
		img.src = imageUrl;
	});
}

export function getFeedColor(iconUrl?: string, feedUrl?: string): Promise<string> {
	if (iconUrl) return extractDominantColor(iconUrl);
	return Promise.resolve(hashToColor(feedUrl || 'default'));
}
