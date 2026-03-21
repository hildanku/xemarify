<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index.js'

	let { action }: { action: string } = $props()

	const variant = $derived.by(() => {
		if (action.startsWith('CREATE')) return 'default'
		if (action.startsWith('UPDATE')) return 'secondary'
		if (action.startsWith('DELETE')) return 'destructive'
		return 'outline'
	})
</script>

<Badge {variant} class="font-mono text-[11px] tracking-wide">{action}</Badge>
