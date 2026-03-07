export function starBurst(x: number, y: number) {
	if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

	const colors = ['#fbbf24', '#f59e0b', '#d97706', '#fcd34d', '#fef3c7'];
	const count = 6;

	for (let i = 0; i < count; i++) {
		const particle = document.createElement('div');
		const angle = (Math.PI * 2 * i) / count;
		const velocity = 30 + Math.random() * 20;
		const size = 4 + Math.random() * 3;

		Object.assign(particle.style, {
			position: 'fixed',
			left: `${x}px`,
			top: `${y}px`,
			width: `${size}px`,
			height: `${size}px`,
			borderRadius: '50%',
			background: colors[i % colors.length],
			pointerEvents: 'none',
			zIndex: '9999',
			transition: 'all 500ms cubic-bezier(0.22, 1, 0.36, 1)',
		});

		document.body.appendChild(particle);

		requestAnimationFrame(() => {
			particle.style.transform = `translate(${Math.cos(angle) * velocity}px, ${Math.sin(angle) * velocity}px)`;
			particle.style.opacity = '0';
		});

		setTimeout(() => particle.remove(), 600);
	}
}

export function confettiBurst() {
	if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

	const colors = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#06b6d4'];
	const count = 30;

	for (let i = 0; i < count; i++) {
		const particle = document.createElement('div');
		const x = window.innerWidth / 2 + (Math.random() - 0.5) * 300;
		const size = 6 + Math.random() * 4;

		Object.assign(particle.style, {
			position: 'fixed',
			left: `${x}px`,
			top: '-10px',
			width: `${size}px`,
			height: `${size * 1.5}px`,
			borderRadius: '2px',
			background: colors[Math.floor(Math.random() * colors.length)],
			pointerEvents: 'none',
			zIndex: '9999',
			transform: `rotate(${Math.random() * 360}deg)`,
			transition: `all ${800 + Math.random() * 400}ms cubic-bezier(0.22, 1, 0.36, 1)`,
		});

		document.body.appendChild(particle);

		requestAnimationFrame(() => {
			particle.style.top = `${window.innerHeight + 20}px`;
			particle.style.left = `${parseFloat(particle.style.left) + (Math.random() - 0.5) * 100}px`;
			particle.style.opacity = '0';
			particle.style.transform = `rotate(${Math.random() * 720}deg)`;
		});

		setTimeout(() => particle.remove(), 1500);
	}
}
