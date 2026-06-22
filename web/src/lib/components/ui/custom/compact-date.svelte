<script lang="ts">
	import { formatDate, formatTime, formatRelative } from '$lib/utils/date'

	let {
		dateString,
		fallback = '—',
	}: {
		dateString: string | null | undefined
		fallback?: string
	} = $props()

	const parsed = $derived(
		dateString
			? {
					date: formatDate(dateString),
					time: formatTime(dateString),
					relative: formatRelative(dateString),
				}
			: null,
	)
</script>

{#if parsed}
	<div class="flex flex-col gap-0.5">
		<span class="text-xs">{parsed.date} {parsed.time}</span>
		<span class="text-xs text-muted-foreground">{parsed.relative}</span>
	</div>
{:else}
	<span class="text-xs text-muted-foreground">{fallback}</span>
{/if}
