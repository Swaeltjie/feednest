interface SwipeOptions {
	onSwipeRight?: () => void;
	onSwipeLeft?: () => void;
	threshold?: number;
}

export function swipeable(node: HTMLElement, options: SwipeOptions) {
	const threshold = options.threshold ?? 80;
	let startX = 0;
	let startY = 0;
	let currentX = 0;
	let isDragging = false;

	const leftIndicator = document.createElement('div');
	const rightIndicator = document.createElement('div');

	leftIndicator.textContent = '\u2713';
	rightIndicator.textContent = '\u2605';

	const baseStyle: Partial<CSSStyleDeclaration> = {
		position: 'absolute',
		top: '0',
		bottom: '0',
		width: '60px',
		display: 'flex',
		alignItems: 'center',
		justifyContent: 'center',
		fontSize: '20px',
		opacity: '0',
		transition: 'opacity 100ms',
		zIndex: '0',
		borderRadius: '8px',
	};

	Object.assign(leftIndicator.style, {
		...baseStyle,
		left: '0',
		background: 'rgba(34, 197, 94, 0.15)',
		color: '#22c55e',
	});

	Object.assign(rightIndicator.style, {
		...baseStyle,
		right: '0',
		background: 'rgba(250, 204, 21, 0.15)',
		color: '#facc15',
	});

	node.style.position = 'relative';
	node.style.overflow = 'hidden';
	node.appendChild(leftIndicator);
	node.appendChild(rightIndicator);

	function onTouchStart(e: TouchEvent) {
		startX = e.touches[0].clientX;
		startY = e.touches[0].clientY;
		currentX = 0;
		isDragging = false;
		node.style.transition = 'none';
	}

	function onTouchMove(e: TouchEvent) {
		const dx = e.touches[0].clientX - startX;
		const dy = e.touches[0].clientY - startY;

		if (!isDragging && Math.abs(dy) > Math.abs(dx)) return;
		isDragging = true;

		currentX = dx;
		const dampened = currentX * 0.6;
		node.style.transform = `translateX(${dampened}px)`;

		leftIndicator.style.opacity = currentX > 20 ? '1' : '0';
		rightIndicator.style.opacity = currentX < -20 ? '1' : '0';

		if (Math.abs(currentX) > threshold) {
			const indicator = currentX > 0 ? leftIndicator : rightIndicator;
			indicator.style.transform = 'scale(1.2)';
		} else {
			leftIndicator.style.transform = 'scale(1)';
			rightIndicator.style.transform = 'scale(1)';
		}
	}

	function onTouchEnd() {
		node.style.transition = `transform var(--duration-snappy) var(--spring-dramatic)`;
		node.style.transform = 'translateX(0)';

		leftIndicator.style.opacity = '0';
		rightIndicator.style.opacity = '0';
		leftIndicator.style.transform = 'scale(1)';
		rightIndicator.style.transform = 'scale(1)';

		if (currentX > threshold) {
			options.onSwipeRight?.();
		} else if (currentX < -threshold) {
			options.onSwipeLeft?.();
		}

		isDragging = false;
	}

	node.addEventListener('touchstart', onTouchStart, { passive: true });
	node.addEventListener('touchmove', onTouchMove, { passive: false });
	node.addEventListener('touchend', onTouchEnd);

	return {
		destroy() {
			node.removeEventListener('touchstart', onTouchStart);
			node.removeEventListener('touchmove', onTouchMove);
			node.removeEventListener('touchend', onTouchEnd);
			leftIndicator.remove();
			rightIndicator.remove();
		},
		update(newOptions: SwipeOptions) {
			options = newOptions;
		}
	};
}
