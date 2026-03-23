<script lang="ts">
	import type { EventItem } from '$lib/types/api'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import MoreHorizontalIcon from '@lucide/svelte/icons/more-horizontal'

	let {
		event,
		onView,
	}: {
		event: EventItem
		onView: (id: string) => void
	} = $props()
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="sm" class="h-8 w-8 p-0" aria-label="Open event actions" {...props}>
				<MoreHorizontalIcon class="h-4 w-4" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-44">
		<DropdownMenu.Item onclick={() => onView(event.id)}>View details</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
