<script lang="ts">
	import { onMount } from 'svelte';
	import { apiClient } from '$lib/api/client';
	import type { Event } from '$lib/types/api';
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as Table from "$lib/components/ui/table/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";

	// State
	let events: Event[] = $state([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	
	// Pagination
	let currentPage = $state(1);
	let limit = $state(50);
	let totalCount = $state(0);
	let totalPages = $derived(Math.ceil(totalCount / limit));

	// Selected event for detail modal
	let selectedEvent = $state<Event | null>(null);
	let dialogOpen = $state(false);

	async function loadEvents() {
		try {
			loading = true;
			error = null;
			const offset = (currentPage - 1) * limit;
			const response = await apiClient.getEvents({ limit, offset });
			events = response.data || [];
			totalCount = response.count || 0;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load events';
			events = [];
			totalCount = 0;
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadEvents();
	});

	function goToPage(page: number) {
		if (page >= 1 && page <= totalPages) {
			currentPage = page;
			loadEvents();
		}
	}

	function formatDate(dateString: string | null): string {
		if (!dateString) return 'N/A';
		return new Date(dateString).toLocaleString('id-ID');
	}

	function getBadgeVariant(level: number | null): "default" | "secondary" | "destructive" | "outline" {
		if (level === null) return "outline";
		if (level >= 12) return "destructive";
		if (level >= 8) return "default";
		if (level >= 4) return "secondary";
		return "outline";
	}

	function openEventDetail(event: Event) {
		selectedEvent = event;
		dialogOpen = true;
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold">Security Events</h1>
			<p class="text-muted-foreground">View and analyze all security events</p>
		</div>
		<Button onclick={() => loadEvents()} disabled={loading}>
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

	<!-- Events Table -->
	<Card.Root>
		<Card.Content class="p-0">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>ID</Table.Head>
						<Table.Head>Time</Table.Head>
						<Table.Head>Hostname</Table.Head>
						<Table.Head>Source IP</Table.Head>
						<Table.Head>Category</Table.Head>
						<Table.Head>Level</Table.Head>
						<Table.Head>Message</Table.Head>
						<Table.Head class="text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#if loading}
						<Table.Row>
							<Table.Cell colspan={8} class="text-center py-8">
								<div class="flex items-center justify-center">
									<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
								</div>
							</Table.Cell>
						</Table.Row>
					{:else if !events || events.length === 0}
						<Table.Row>
							<Table.Cell colspan={8} class="text-center py-8 text-muted-foreground">
								No events found
							</Table.Cell>
						</Table.Row>
					{:else}
						{#each events || [] as event}
							<Table.Row class="cursor-pointer" onclick={() => openEventDetail(event)}>
								<Table.Cell class="font-medium">{event.id}</Table.Cell>
								<Table.Cell class="text-xs">{formatDate(event.event_time)}</Table.Cell>
								<Table.Cell>{event.hostname || 'N/A'}</Table.Cell>
								<Table.Cell class="font-mono text-xs">{event.source_ip || 'N/A'}</Table.Cell>
								<Table.Cell>
									{#if event.category}
										<Badge variant="secondary">{event.category}</Badge>
									{:else}
										N/A
									{/if}
								</Table.Cell>
								<Table.Cell>
									{#if event.level !== null}
										<Badge variant={getBadgeVariant(event.level)}>
											Level {event.level}
										</Badge>
									{:else}
										N/A
									{/if}
								</Table.Cell>
								<Table.Cell class="max-w-md truncate">{event.message}</Table.Cell>
								<Table.Cell class="text-right">
									<Button 
										size="sm"
										variant="ghost"
										onclick={(e) => { e.stopPropagation(); openEventDetail(event); }}
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
		{#if !loading && events.length > 0}
			<Card.Footer class="flex items-center justify-between">
				<div class="text-sm text-muted-foreground">
					Showing <span class="font-medium">{(currentPage - 1) * limit + 1}</span> to 
					<span class="font-medium">{Math.min(currentPage * limit, totalCount)}</span> of 
					<span class="font-medium">{totalCount}</span> events
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

<!-- Event Detail Dialog -->
<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Content class="max-w-3xl max-h-[90vh] overflow-y-auto">
		{#if selectedEvent}
			<Dialog.Header>
				<Dialog.Title>Event Details</Dialog.Title>
				<Dialog.Description>Event ID: {selectedEvent.id}</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-4">
				<div class="grid grid-cols-2 gap-4">
					<div>
						<p class="text-sm font-medium text-muted-foreground">Event ID</p>
						<p class="text-base font-mono">{selectedEvent.id}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Event Time</p>
						<p class="text-base">{formatDate(selectedEvent.event_time)}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Hostname</p>
						<p class="text-base">{selectedEvent.hostname || 'N/A'}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Source IP</p>
						<p class="text-base font-mono">{selectedEvent.source_ip || 'N/A'}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Category</p>
						<p class="text-base">{selectedEvent.category || 'N/A'}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Severity Level</p>
						{#if selectedEvent.level !== null}
							<Badge variant={getBadgeVariant(selectedEvent.level)}>
								Level {selectedEvent.level}
							</Badge>
						{:else}
							<p class="text-base">N/A</p>
						{/if}
					</div>
				</div>

				<div>
					<p class="text-sm font-medium text-muted-foreground mb-2">Message</p>
					<Card.Root>
						<Card.Content class="pt-4">
							<p class="text-base">{selectedEvent.message}</p>
						</Card.Content>
					</Card.Root>
				</div>

				{#if selectedEvent.raw}
					<div>
						<p class="text-sm font-medium text-muted-foreground mb-2">Raw Log</p>
						<Card.Root>
							<Card.Content class="pt-4">
								<pre class="text-sm overflow-x-auto">{selectedEvent.raw}</pre>
							</Card.Content>
						</Card.Root>
					</div>
				{/if}

				{#if selectedEvent.normalized && Object.keys(selectedEvent.normalized).length > 0}
					<div>
						<p class="text-sm font-medium text-muted-foreground mb-2">Normalized Data</p>
						<Card.Root>
							<Card.Content class="pt-4">
								<pre class="text-sm overflow-x-auto">{JSON.stringify(selectedEvent.normalized, null, 2)}</pre>
							</Card.Content>
						</Card.Root>
					</div>
				{/if}
			</div>
			<Dialog.Footer>
				<Button variant="outline" onclick={() => dialogOpen = false}>Close</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
