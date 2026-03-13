<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index.js'

	let { status }: { status: string } = $props()

	const variants: Record<string, 'default' | 'secondary' | 'outline' | 'destructive'> = {
		new: 'secondary',
		acknowledged: 'outline',
		closed: 'default',
	}

	const label = $derived(status ? status.charAt(0).toUpperCase() + status.slice(1) : 'Unknown')
	const variant = $derived(variants[status] ?? 'secondary')
</script>

<Badge {variant} class="capitalize">{label}</Badge>
