<script lang="ts">
	import { onMount } from 'svelte';
	import { apiClient } from '$lib/api/client';
	import type { Alert, AlertStatus } from '$lib/types/api';
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as Table from "$lib/components/ui/table/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import * as Select from "$lib/components/ui/select/index.js";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";

	// State
	let alerts = $state<Alert[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	
	// Filters
	let statusFilter = $state<AlertStatus | 'all'>('all');
	
	// Pagination
	let currentPage = $state(1);
	let limit = $state(50);
	let totalCount = $state(0);
	let totalPages = $derived(Math.ceil(totalCount / limit));

	async function loadAlerts() {
		try {
			loading = true;
			error = null;
			const offset = (currentPage - 1) * limit;
			const params: any = { limit, offset };
			
			if (statusFilter !== 'all') {
				params.status = statusFilter;
			}
			
			const response = await apiClient.getAlerts(params);
			alerts = response.data;
			totalCount = response.count;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load alerts';
		} finally {
			loading = false;
		}
	}

	async function updateAlertStatus(alertId: number, newStatus: AlertStatus) {
		try {
			await apiClient.updateAlertStatus(alertId, { status: newStatus });
			await loadAlerts(); // Reload data
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update alert status';
		}
	}

	onMount(() => {
		loadAlerts();
	});

	function goToPage(page: number) {
		if (page >= 1 && page <= totalPages) {
			currentPage = page;
			loadAlerts();
		}
	}

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleString('id-ID');
	}

	function getBadgeVariant(level: number): "default" | "secondary" | "destructive" | "outline" {
		if (level >= 12) return "destructive";
		if (level >= 8) return "default";
		if (level >= 4) return "secondary";
		return "outline";
	}

	function getStatusVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
		if (status === 'new') return "destructive";
		if (status === 'acknowledged') return "default";
		if (status === 'resolved') return "secondary";
		return "outline";
	}

	function handleStatusFilterChange(value: any) {
		statusFilter = value.value as AlertStatus | 'all';
		currentPage = 1;
		loadAlerts();
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold">Security Alerts</h1>
			<p class="text-muted-foreground">Review and manage security alerts</p>
		</div>
		<div class="flex gap-2">
			<Select.Root onSelectedChange={handleStatusFilterChange}>
				<Select.Trigger class="w-[180px]">
					<Select.Value placeholder="Filter by status" />
				</Select.Trigger>
				<Select.Content>
					<Select.Item value="all">All Statuses</Select.Item>
					<Select.Item value="new">New</Select.Item>
					<Select.Item value="acknowledged">Acknowledged</Select.Item>
					<Select.Item value="investigating">Investigating</Select.Item>
					<Select.Item value="resolved">Resolved</Select.Item>
					<Select.Item value="false_positive">False Positive</Select.Item>
				</Select.Content>
			</Select.Root>
			<Button onclick={() => loadAlerts()} disabled={loading}>
				<RefreshCwIcon class="h-4 w-4 mr-2" />
				Refresh
			</Button>
		</div>
	</div>

	{#if error}
		<Card.Root class="border-destructive">
			<Card.Content class="pt-6">
				<p class="text-destructive">{error}</p>
			</Card.Content>
		</Card.Root>
	{/if}

	<!-- Alerts Table -->
	<Card.Root>
		<Card.Content class="p-0">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>ID</Table.Head>
						<Table.Head>Created At</Table.Head>
						<Table.Head>Event ID</Table.Head>
						<Table.Head>Level</Table.Head>
						<Table.Head>Status</Table.Head>
						<Table.Head>Rule ID</Table.Head>
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
					{:else if alerts.length === 0}
						<Table.Row>
							<Table.Cell colspan={7} class="text-center py-8 text-muted-foreground">
								No alerts found
							</Table.Cell>
						</Table.Row>
					{:else}
						{#each alerts as alert}
							<Table.Row>
								<Table.Cell class="font-medium">{alert.id}</Table.Cell>
								<Table.Cell>{formatDate(alert.created_at)}</Table.Cell>
								<Table.Cell>
									<a href="/dashboard/events/{alert.event_id}" class="text-blue-600 hover:underline">
										{alert.event_id}
									</a>
								</Table.Cell>
								<Table.Cell>
									<Badge variant={getBadgeVariant(alert.level)}>
										Level {alert.level}
									</Badge>
								</Table.Cell>
								<Table.Cell>
									<Badge variant={getStatusVariant(alert.status)}>
										{alert.status}
									</Badge>
								</Table.Cell>
								<Table.Cell class="font-mono text-xs">{alert.rule_id.slice(0, 8)}...</Table.Cell>
								<Table.Cell class="text-right">
									<div class="flex justify-end gap-2">
										{#if alert.status === 'new'}
											<Button 
												size="sm" 
												variant="outline"
												onclick={() => updateAlertStatus(alert.id, 'acknowledged')}
											>
												Acknowledge
											</Button>
										{/if}
										{#if alert.status === 'acknowledged' || alert.status === 'investigating'}
											<Button 
												size="sm" 
												variant="outline"
												onclick={() => updateAlertStatus(alert.id, 'resolved')}
											>
												Resolve
											</Button>
										{/if}
									</div>
								</Table.Cell>
							</Table.Row>
						{/each}
					{/if}
				</Table.Body>
			</Table.Root>
		</Card.Content>

		<!-- Pagination -->
		{#if !loading && alerts.length > 0}
			<Card.Footer class="flex items-center justify-between">
				<div class="text-sm text-muted-foreground">
					Showing <span class="font-medium">{(currentPage - 1) * limit + 1}</span> to 
					<span class="font-medium">{Math.min(currentPage * limit, totalCount)}</span> of 
					<span class="font-medium">{totalCount}</span> alerts
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
