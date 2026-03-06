<script lang="ts">
	import { onMount } from 'svelte';
	import { apiClient } from '$lib/api/client';
	import type { Agent } from '$lib/types/api';
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as Table from "$lib/components/ui/table/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
	import CircleIcon from "@lucide/svelte/icons/circle";

	// State
	let agents = $state<Agent[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	
	// Pagination
	let currentPage = $state(1);
	let limit = $state(50);
	let totalCount = $state(0);
	let totalPages = $derived(Math.ceil(totalCount / limit));

	// Selected agent for detail modal
	let selectedAgent = $state<Agent | null>(null);
	let dialogOpen = $state(false);

	// Stats
	let onlineCount = $state(0);
	let offlineCount = $state(0);

	async function loadAgents() {
		try {
			loading = true;
			error = null;
			const offset = (currentPage - 1) * limit;
			const response = await apiClient.getAgents({ limit, offset });
			agents = response.data;
			totalCount = response.count;

			// Calculate online/offline counts
			const now = new Date();
			const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
			onlineCount = agents.filter(agent => 
				agent.last_seen_at && new Date(agent.last_seen_at) > fiveMinutesAgo
			).length;
			offlineCount = agents.length - onlineCount;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load agents';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadAgents();
		// Auto-refresh every 30 seconds
		const interval = setInterval(loadAgents, 30000);
		return () => clearInterval(interval);
	});

	function goToPage(page: number) {
		if (page >= 1 && page <= totalPages) {
			currentPage = page;
			loadAgents();
		}
	}

	function formatDate(dateString: string | null): string {
		if (!dateString) return 'Never';
		return new Date(dateString).toLocaleString('id-ID');
	}

	function isOnline(lastSeenAt: string | null): boolean {
		if (!lastSeenAt) return false;
		const now = new Date();
		const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
		return new Date(lastSeenAt) > fiveMinutesAgo;
	}

	function getTimeSince(dateString: string | null): string {
		if (!dateString) return 'Never';
		const now = new Date();
		const then = new Date(dateString);
		const seconds = Math.floor((now.getTime() - then.getTime()) / 1000);
		
		if (seconds < 60) return `${seconds}s ago`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
		if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
		return `${Math.floor(seconds / 86400)}d ago`;
	}

	function openAgentDetail(agent: Agent) {
		selectedAgent = agent;
		dialogOpen = true;
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold">Agents</h1>
			<p class="text-muted-foreground">Monitor connected security agents</p>
		</div>
		<Button onclick={() => loadAgents()} disabled={loading}>
			<RefreshCwIcon class="h-4 w-4 mr-2" />
			Refresh
		</Button>
	</div>

	{#if error}
		<Card.Root class="border-destructive">
			<Card.Content class="pt-6">
				<p class="text-destructive">{error}</p>
			</Card.Content>
		</Card.Root>
	{/if}

	<!-- Stats -->
	<div class="grid gap-4 md:grid-cols-3">
		<Card.Root>
			<Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
				<Card.Title class="text-sm font-medium">Total Agents</Card.Title>
			</Card.Header>
			<Card.Content>
				<div class="text-2xl font-bold">{totalCount}</div>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
				<Card.Title class="text-sm font-medium">Online</Card.Title>
				<CircleIcon class="h-4 w-4 text-green-600 fill-green-600" />
			</Card.Header>
			<Card.Content>
				<div class="text-2xl font-bold text-green-600">{onlineCount}</div>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
				<Card.Title class="text-sm font-medium">Offline</Card.Title>
				<CircleIcon class="h-4 w-4 text-gray-400 fill-gray-400" />
			</Card.Header>
			<Card.Content>
				<div class="text-2xl font-bold text-gray-600">{offlineCount}</div>
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Agents Table -->
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
					{#if loading}
						<Table.Row>
							<Table.Cell colspan={7} class="text-center py-8">
								<div class="flex items-center justify-center">
									<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
								</div>
							</Table.Cell>
						</Table.Row>
					{:else if agents.length === 0}
						<Table.Row>
							<Table.Cell colspan={7} class="text-center py-8 text-muted-foreground">
								No agents found
							</Table.Cell>
						</Table.Row>
					{:else}
						{#each agents as agent}
							<Table.Row class="cursor-pointer" onclick={() => openAgentDetail(agent)}>
								<Table.Cell>
									{#if isOnline(agent.last_seen_at)}
										<div class="flex items-center gap-2">
											<CircleIcon class="h-3 w-3 text-green-600 fill-green-600" />
											<Badge variant="outline" class="text-green-600 border-green-600">Online</Badge>
										</div>
									{:else}
										<div class="flex items-center gap-2">
											<CircleIcon class="h-3 w-3 text-gray-400 fill-gray-400" />
											<Badge variant="outline" class="text-gray-600">Offline</Badge>
										</div>
									{/if}
								</Table.Cell>
								<Table.Cell class="font-medium">{agent.name}</Table.Cell>
								<Table.Cell>{agent.hostname || 'N/A'}</Table.Cell>
								<Table.Cell class="font-mono text-xs">{agent.ip_address || 'N/A'}</Table.Cell>
								<Table.Cell>
									<Badge variant="secondary">{agent.version || 'N/A'}</Badge>
								</Table.Cell>
								<Table.Cell class="text-xs">
									<div>{formatDate(agent.last_seen_at)}</div>
									{#if agent.last_seen_at}
										<div class="text-muted-foreground">{getTimeSince(agent.last_seen_at)}</div>
									{/if}
								</Table.Cell>
								<Table.Cell class="text-right">
									<Button 
										size="sm"
										variant="ghost"
										onclick={(e) => { e.stopPropagation(); openAgentDetail(agent); }}
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

		<!-- Pagination -->
		{#if !loading && agents.length > 0}
			<Card.Footer class="flex items-center justify-between">
				<div class="text-sm text-muted-foreground">
					Showing <span class="font-medium">{(currentPage - 1) * limit + 1}</span> to 
					<span class="font-medium">{Math.min(currentPage * limit, totalCount)}</span> of 
					<span class="font-medium">{totalCount}</span> agents
				</div>
				<div class="flex gap-2">
					<Button
						variant="outline"
						size="sm"
						onclick={() => goToPage(currentPage - 1)}
						disabled={currentPage === 1}
					>
						Previous
					</Button>
					<div class="flex items-center px-3 text-sm">
						Page {currentPage} of {totalPages}
					</div>
					<Button
						variant="outline"
						size="sm"
						onclick={() => goToPage(currentPage + 1)}
						disabled={currentPage === totalPages}
					>
						Next
					</Button>
				</div>
			</Card.Footer>
		{/if}
	</Card.Root>
</div>

<!-- Agent Detail Dialog -->
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
						<Badge variant="outline" class="text-green-600 border-green-600">Online</Badge>
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
						<p class="text-base font-medium">{selectedAgent.name}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Hostname</p>
						<p class="text-base">{selectedAgent.hostname || 'N/A'}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">IP Address</p>
						<p class="text-base font-mono">{selectedAgent.ip_address || 'N/A'}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Version</p>
						<Badge variant="secondary">{selectedAgent.version || 'N/A'}</Badge>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Created At</p>
						<p class="text-sm">{formatDate(selectedAgent.created_at)}</p>
					</div>
					<div class="col-span-2">
						<p class="text-sm font-medium text-muted-foreground">Last Seen</p>
						<p class="text-base">{formatDate(selectedAgent.last_seen_at)}</p>
						{#if selectedAgent.last_seen_at}
							<p class="text-sm text-muted-foreground">{getTimeSince(selectedAgent.last_seen_at)}</p>
						{/if}
					</div>
				</div>
			</div>
			<Dialog.Footer>
				<Button variant="outline" onclick={() => dialogOpen = false}>Close</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
