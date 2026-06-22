<script lang="ts">
	import type { Alert, AlertStatus } from '$lib/types/api'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import MoreHorizontalIcon from '@lucide/svelte/icons/more-horizontal'

	import { auth } from '$lib/auth/session'

	let {
		alert,
		onView,
		onStatus,
	}: {
		alert: Alert
		onView: (id: string) => void
		onStatus: (id: string, status: AlertStatus) => void
	} = $props()

	const isViewer = $derived($auth.user?.role === 'VIEWER')
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="sm" class="h-8 w-8 p-0" aria-label="Open alert actions" {...props}>
				<MoreHorizontalIcon class="h-4 w-4" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-44">
		<DropdownMenu.Item onclick={() => onView(alert.id)}>View events</DropdownMenu.Item>
		{#if !isViewer}
			<DropdownMenu.Item
				disabled={alert.status === 'acknowledged' || alert.status === 'closed'}
				onclick={() => onStatus(alert.id, 'acknowledged')}
			>
				Acknowledge alert
			</DropdownMenu.Item>
			<DropdownMenu.Item
				disabled={alert.status === 'closed'}
				onclick={() => onStatus(alert.id, 'closed')}
			>
				Close alert
			</DropdownMenu.Item>
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
