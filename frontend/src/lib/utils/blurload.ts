export function blurUp(node: HTMLImageElement) {
	if (node.complete) return { destroy() {} };

	node.style.filter = 'blur(10px)';
	node.style.transform = 'scale(1.05)';
	node.style.transition = 'filter var(--duration-smooth) var(--spring-smooth), transform var(--duration-smooth) var(--spring-smooth)';

	function onLoad() {
		node.style.filter = 'blur(0)';
		node.style.transform = 'scale(1)';
	}

	function onError() {
		node.style.filter = 'blur(0)';
		node.style.transform = 'scale(1)';
	}

	node.addEventListener('load', onLoad);
	node.addEventListener('error', onError);

	return {
		destroy() {
			node.removeEventListener('load', onLoad);
			node.removeEventListener('error', onError);
		}
	};
}
