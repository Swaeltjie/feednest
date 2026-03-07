export function parallax(node: HTMLElement, options?: { rate?: number }) {
	const rate = options?.rate ?? 0.3;

	if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
		return { destroy() {} };
	}

	const img = node.querySelector('img') as HTMLElement | null;
	if (!img) return { destroy() {} };

	node.style.overflow = 'hidden';
	img.style.willChange = 'transform';

	function onScroll() {
		const rect = node.getBoundingClientRect();
		const viewportHeight = window.innerHeight;
		if (rect.bottom < 0 || rect.top > viewportHeight) return;

		const centerOffset = rect.top - viewportHeight / 2;
		const translateY = centerOffset * rate;
		img!.style.transform = `translateY(${translateY}px) scale(1.1)`;
	}

	window.addEventListener('scroll', onScroll, { passive: true });
	onScroll();

	return {
		destroy() {
			window.removeEventListener('scroll', onScroll);
		}
	};
}

export function magneticHover(node: HTMLElement, options?: { strength?: number }) {
	const strength = options?.strength ?? 5;

	if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
		return { destroy() {} };
	}

	const img = node.querySelector('img') as HTMLElement | null;
	if (!img) return { destroy() {} };

	function onMouseMove(e: MouseEvent) {
		const rect = node.getBoundingClientRect();
		const x = ((e.clientX - rect.left) / rect.width - 0.5) * strength;
		const y = ((e.clientY - rect.top) / rect.height - 0.5) * strength;
		img!.style.transform = `translate(${x}px, ${y}px) scale(1.05)`;
	}

	function onMouseLeave() {
		img!.style.transition = `transform var(--duration-snappy) var(--spring-snappy)`;
		img!.style.transform = 'translate(0, 0) scale(1)';
	}

	function onMouseEnter() {
		img!.style.transition = 'none';
	}

	node.addEventListener('mouseenter', onMouseEnter);
	node.addEventListener('mousemove', onMouseMove);
	node.addEventListener('mouseleave', onMouseLeave);

	return {
		destroy() {
			node.removeEventListener('mouseenter', onMouseEnter);
			node.removeEventListener('mousemove', onMouseMove);
			node.removeEventListener('mouseleave', onMouseLeave);
		}
	};
}
