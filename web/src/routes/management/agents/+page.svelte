<script lang="ts">
	import type { Agent } from '$lib/types/api'
	import type { ApiResponseWithMetadata } from '$lib/client'
	import { Button } from '$lib/components/ui/button/index.js'
	import * as Card from '$lib/components/ui/card/index.js'
	import * as Table from '$lib/components/ui/table/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw'
	import CircleIcon from '@lucide/svelte/icons/circle'
	import { createQuery } from '@tanstack/svelte-query'
	import { clientFetch } from '$lib/client'

	function formatDate(dateString: string | null): string {
		if (!dateString) return 'Never'
		return new Date(dateString).toLocaleString('id-ID')
	}

	function isOnline(lastSeenAt: string | null): boolean {
		if (!lastSeenAt) return false
		const now = new Date()
		const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000)
		return new Date(lastSeenAt) > fiveMinutesAgo
	}

	function getTimeSince(dateString: string | null): string {
		if (!dateString) return 'Never'
		const now = new Date()
		const then = new Date(dateString)
		const seconds = Math.floor((now.getTime() - then.getTime()) / 1000)

		if (seconds < 60) return `${seconds}s ago`
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
		if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
		return `${Math.floor(seconds / 86400)}d ago`
	}

	let dialogOpen = $state(false)
	let selectedAgent = $state<Agent | null>(null)

	function openAgentDetail(agent: Agent) {
		selectedAgent = agent
		dialogOpen = true
	}

	const agentsQuery = createQuery<ApiResponseWithMetadata<Agent[]>>(() => ({
		queryKey: ['agents'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Agent[]>>(
				'http://localhost:8089/api/v1/agents',
				{ method: 'GET' },
			),
	}))

	const agents = $derived(agentsQuery.data?.data.items ?? [])
	const onlineCount = $derived(
		agents.filter((a) => isOnline(a.last_seen_at)).length,
	)
	const offlineCount = $derived(agents.length - onlineCount)
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold">Agents</h1>
			<p class="text-muted-foreground">Monitor connected security agents</p>
		</div>
		<Button onclick={() => agentsQuery.refetch()}>
			<RefreshCwIcon class="h-4 w-4 mr-2" />
			Refresh
		</Button>
	</div>
	<Card.Root>
		<Card.Content class="p-0">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>Status</Table.Head>
						<Table.Head>Name</Table.Head>
						<Table.Head>Hostname</Table.Head>
						<Table.Head>IP Address</Table.Head>
						<Table.Head>Version</Table.Head>
						<Table.Head>Last Seen</Table.Head>
						<Table.Head class="text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#if agentsQuery.isPending}
						<Table.Row>
							<Table.Cell colspan={7} class="text-center py-8">
								<div class="flex items-center justify-center">
									<div
										class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"
									></div>
								</div>
							</Table.Cell>
						</Table.Row>
					{:else if agents.length === 0}
						<Table.Row>
							<Table.Cell
								colspan={7}
								class="text-center py-8 text-muted-foreground"
							>
								No agents found
							</Table.Cell>
						</Table.Row>
					{:else}
						{#each agents as agent}
							<Table.Row
								class="cursor-pointer"
								onclick={() => openAgentDetail(agent)}
							>
								<Table.Cell>
									{#if isOnline(agent.last_seen_at)}
										<div class="flex items-center gap-2">
											<CircleIcon
												class="h-3 w-3 text-green-600 fill-green-600"
											/>
											<Badge
												variant="outline"
												class="text-green-600 border-green-600">Online</Badge
											>
										</div>
									{:else}
										<div class="flex items-center gap-2">
											<CircleIcon class="h-3 w-3 text-gray-400 fill-gray-400" />
											<Badge variant="outline" class="text-gray-600"
												>Offline</Badge
											>
										</div>
									{/if}
								</Table.Cell>
								<Table.Cell class="font-medium">{agent.name}</Table.Cell>
								<Table.Cell>{agent.hostname || 'N/A'}</Table.Cell>
								<Table.Cell class="font-mono text-xs"
									>{agent.ip_address || 'N/A'}</Table.Cell
								>
								<Table.Cell>
									<Badge variant="secondary">{agent.version || 'N/A'}</Badge>
								</Table.Cell>
								<Table.Cell class="text-xs">
									<div>{formatDate(agent.last_seen_at)}</div>
									{#if agent.last_seen_at}
										<div class="text-muted-foreground">
											{getTimeSince(agent.last_seen_at)}
										</div>
									{/if}
								</Table.Cell>
								<Table.Cell class="text-right">
									<Button
										size="sm"
										variant="ghost"
										onclick={(e) => {
											e.stopPropagation()
											openAgentDetail(agent)
										}}
									>
										View
									</Button>
								</Table.Cell>
							</Table.Row>
						{/each}
					{/if}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
</div>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Content class="max-w-2xl">
		{#if selectedAgent}
			<Dialog.Header>
				<Dialog.Title>{selectedAgent.name}</Dialog.Title>
				<Dialog.Description>Agent Details</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-4">
				<div class="flex items-center gap-2">
					{#if isOnline(selectedAgent.last_seen_at)}
						<CircleIcon class="h-4 w-4 text-green-600 fill-green-600" />
						<Badge variant="outline" class="text-green-600 border-green-600"
							>Online</Badge
						>
					{:else}
						<CircleIcon class="h-4 w-4 text-gray-400 fill-gray-400" />
						<Badge variant="outline" class="text-gray-600">Offline</Badge>
					{/if}
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div>
						<p class="text-sm font-medium text-muted-foreground">Agent ID</p>
						<p class="text-sm font-mono">{selectedAgent.id}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Name</p>
						<p class="text-base font-medium">
							{selectedAgent.name}
						</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Hostname</p>
						<p class="text-base">
							{selectedAgent.hostname || 'N/A'}
						</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">IP Address</p>
						<p class="text-base font-mono">
							{selectedAgent.ip_address || 'N/A'}
						</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Version</p>
						<Badge variant="secondary">{selectedAgent.version || 'N/A'}</Badge>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Created At</p>
						<p class="text-sm">
							{formatDate(selectedAgent.created_at)}
						</p>
					</div>
					<div class="col-span-2">
						<p class="text-sm font-medium text-muted-foreground">Last Seen</p>
						<p class="text-base">
							{formatDate(selectedAgent.last_seen_at)}
						</p>
						{#if selectedAgent.last_seen_at}
							<p class="text-sm text-muted-foreground">
								{getTimeSince(selectedAgent.last_seen_at)}
							</p>
						{/if}
					</div>
				</div>
			</div>
			<Dialog.Footer>
				<Button variant="outline" onclick={() => (dialogOpen = false)}
					>Close</Button
				>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
