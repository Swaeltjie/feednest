<script lang="ts">
	let { value }: { value: number } = $props();

	let displayValue = $state(value);

	$effect(() => {
		if (value === displayValue) return;
		const step = value > displayValue ? 1 : -1;
		const diff = Math.abs(value - displayValue);
		const intervalMs = Math.max(20, Math.min(80, 400 / diff));
		const interval = setInterval(() => {
			displayValue += step;
			if (displayValue === value) clearInterval(interval);
		}, intervalMs);
		return () => clearInterval(interval);
	});
</script>

<span class="inline-flex tabular-nums">
	{displayValue}
</span>
