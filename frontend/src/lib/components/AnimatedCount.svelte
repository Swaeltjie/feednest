<script lang="ts">
	import { untrack } from 'svelte';

	let { value }: { value: number } = $props();

	let displayValue = $state(value);

	$effect(() => {
		const target = value;
		const current = untrack(() => displayValue);
		if (target === current) return;
		const step = target > current ? 1 : -1;
		const diff = Math.abs(target - current);
		const intervalMs = Math.max(20, Math.min(80, 400 / diff));
		const interval = setInterval(() => {
			displayValue += step;
			if (displayValue === target) clearInterval(interval);
		}, intervalMs);
		return () => clearInterval(interval);
	});
</script>

<span class="inline-flex tabular-nums">
	{displayValue}
</span>
