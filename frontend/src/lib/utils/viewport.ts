export function viewportFadeIn(node: HTMLElement, options?: { delay?: number }) {
	const delay = options?.delay ?? 0;

	if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
		node.style.opacity = '1';
		node.style.transform = 'none';
		return { destroy() {} };
	}

	node.style.opacity = '0';
	node.style.transform = 'translateY(12px)';
	node.style.transition = `opacity var(--duration-smooth) var(--spring-smooth) ${delay}ms, transform var(--duration-smooth) var(--spring-smooth) ${delay}ms`;

	const observer = new IntersectionObserver(
		(entries) => {
			entries.forEach((entry) => {
				if (entry.isIntersecting) {
					node.style.opacity = '1';
					node.style.transform = 'translateY(0)';
					observer.unobserve(node);
				}
			});
		},
		{ threshold: 0.1 }
	);

	observer.observe(node);

	return {
		destroy() {
			observer.disconnect();
		}
	};
}
