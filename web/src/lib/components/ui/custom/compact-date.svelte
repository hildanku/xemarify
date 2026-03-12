<script lang="ts">
	let {
		dateString,
		fallback = '—',
	}: {
		dateString: string | null | undefined
		fallback?: string
	} = $props()

	function format(ds: string): {
		date: string
		time: string
		relative: string
	} {
		const d = new Date(ds)
		const date = d.toLocaleDateString('id-ID', {
			day: '2-digit',
			month: 'short',
			year: 'numeric',
		})
		const time = d.toLocaleTimeString('id-ID', {
			hour: '2-digit',
			minute: '2-digit',
		})

		const diffMs = Date.now() - d.getTime()
		const secs = Math.floor(diffMs / 1000)
		let relative = ''
		if (secs < 60) relative = `${secs}s ago`
		else if (secs < 3600) relative = `${Math.floor(secs / 60)}m ago`
		else if (secs < 86400) relative = `${Math.floor(secs / 3600)}h ago`
		else relative = `${Math.floor(secs / 86400)}d ago`

		return { date, time, relative }
	}

	const parsed = $derived(dateString ? format(dateString) : null)
</script>

{#if parsed}
	<div class="flex flex-col gap-0.5">
		<span class="text-xs">{parsed.date} {parsed.time}</span>
		<span class="text-xs text-muted-foreground">{parsed.relative}</span>
	</div>
{:else}
	<span class="text-xs text-muted-foreground">{fallback}</span>
{/if}
