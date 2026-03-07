<script lang="ts">
	let { value }: { value: number } = $props();

	let displayValue = $state(value);

	$effect(() => {
		const target = value;
		if (target === displayValue) return;
		const step = target > displayValue ? 1 : -1;
		const diff = Math.abs(target - displayValue);
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
