<script lang="ts">
	import { onMount } from 'svelte';
	import { apiClient } from '$lib/api/client';
	import type { Alert } from '$lib/types/api';
	import { getSeverityLevel, ALERT_STATUS_COLORS } from '$lib/types/api';
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import FileTextIcon from "@lucide/svelte/icons/file-text";
	import BellIcon from "@lucide/svelte/icons/bell";
	import ServerIcon from "@lucide/svelte/icons/server";
	import AlertTriangleIcon from "@lucide/svelte/icons/alert-triangle";

	// State
	let loading = $state(true);
	let error = $state<string | null>(null);
	
	// Stats
	let totalEvents = $state(0);
	let totalAlerts = $state(0);
	let newAlerts = $state(0);
	let criticalAlerts = $state(0);
	let recentAlerts = $state<Alert[]>([]);
	let onlineAgents = $state(0);
	let totalAgents = $state(0);

	async function loadDashboardData() {
		try {
			loading = true;
			error = null;

			const [eventsRes, alertsRes, newAlertsRes, criticalAlertsRes, agentsRes] = await Promise.all([
				apiClient.getEvents({ limit: 1, offset: 0 }),
				apiClient.getAlerts({ limit: 1, offset: 0 }),
				apiClient.getAlerts({ status: 'new', limit: 1, offset: 0 }),
				apiClient.getAlerts({ limit: 10, offset: 0 }),
				apiClient.getAgents({ limit: 100, offset: 0 })
			]);

			totalEvents = eventsRes.count;
			totalAlerts = alertsRes.count;
			newAlerts = newAlertsRes.count;
			criticalAlerts = criticalAlertsRes.data.filter(a => a.level >= 10).length;
			recentAlerts = criticalAlertsRes.data.slice(0, 5);
			
			const now = new Date();
			const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
			totalAgents = agentsRes.count;
			onlineAgents = agentsRes.data.filter(agent => 
				agent.last_seen_at && new Date(agent.last_seen_at) > fiveMinutesAgo
			).length;

		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load dashboard data';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadDashboardData();
		const interval = setInterval(loadDashboardData, 30000);
		return () => clearInterval(interval);
	});

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
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold">Dashboard</h1>
			<p class="text-muted-foreground">Security Information and Event Management</p>
		</div>
		<Button onclick={() => loadDashboardData()} disabled={loading}>
			{loading ? 'Refreshing...' : 'Refresh'}
		</Button>
	</div>

	{#if error}
		<Card.Root class="border-destructive">
			<Card.Content class="pt-6">
				<p class="text-destructive">{error}</p>
			</Card.Content>
		</Card.Root>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
		</div>
	{:else}
		<!-- Stats Grid -->
		<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
			<!-- Total Events Card -->
			<Card.Root>
				<Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
					<Card.Title class="text-sm font-medium">Total Events</Card.Title>
					<FileTextIcon class="h-4 w-4 text-muted-foreground" />
				</Card.Header>
				<Card.Content>
					<div class="text-2xl font-bold">{totalEvents.toLocaleString()}</div>
					<p class="text-xs text-muted-foreground">All security events</p>
				</Card.Content>
			</Card.Root>

			<!-- Total Alerts Card -->
			<Card.Root>
				<Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
					<Card.Title class="text-sm font-medium">Total Alerts</Card.Title>
					<BellIcon class="h-4 w-4 text-muted-foreground" />
				</Card.Header>
				<Card.Content>
					<div class="text-2xl font-bold">{totalAlerts.toLocaleString()}</div>
					<p class="text-xs text-muted-foreground">Security notifications</p>
				</Card.Content>
			</Card.Root>

			<!-- New Alerts Card -->
			<Card.Root>
				<Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
					<Card.Title class="text-sm font-medium">New Alerts</Card.Title>
					<AlertTriangleIcon class="h-4 w-4 text-destructive" />
				</Card.Header>
				<Card.Content>
					<div class="text-2xl font-bold text-destructive">{newAlerts.toLocaleString()}</div>
					<p class="text-xs text-muted-foreground">Requires attention</p>
				</Card.Content>
			</Card.Root>

			<!-- Online Agents Card -->
			<Card.Root>
				<Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
					<Card.Title class="text-sm font-medium">Online Agents</Card.Title>
					<ServerIcon class="h-4 w-4 text-muted-foreground" />
				</Card.Header>
				<Card.Content>
					<div class="text-2xl font-bold">
						<span class="text-green-600">{onlineAgents}</span>
						<span class="text-muted-foreground">/{totalAgents}</span>
					</div>
					<p class="text-xs text-muted-foreground">Connected sources</p>
				</Card.Content>
			</Card.Root>
		</div>

		<!-- Recent Critical Alerts -->
		<Card.Root>
			<Card.Header>
				<Card.Title>Recent Critical Alerts</Card.Title>
				<Card.Description>Latest high-priority security alerts</Card.Description>
			</Card.Header>
			<Card.Content>
				{#if recentAlerts.length === 0}
					<p class="text-center text-muted-foreground py-8">No critical alerts found</p>
				{:else}
					<div class="space-y-4">
						{#each recentAlerts as alert}
							<div class="flex items-center justify-between border rounded-lg p-4">
								<div class="flex-1 space-y-1">
									<div class="flex items-center gap-2">
										<Badge variant={getBadgeVariant(alert.level)}>
											Level {alert.level}
										</Badge>
										<Badge variant={getStatusVariant(alert.status)}>
											{alert.status}
										</Badge>
									</div>
									<p class="text-sm text-muted-foreground">
										Alert #{alert.id} • Event #{alert.event_id} • Rule: {alert.rule_id.slice(0, 8)}...
									</p>
									<p class="text-xs text-muted-foreground">
										{formatDate(alert.created_at)}
									</p>
								</div>
								<Button href="/dashboard/alerts/{alert.id}" variant="outline" size="sm">
									View Details
								</Button>
							</div>
						{/each}
					</div>
				{/if}
			</Card.Content>
		</Card.Root>

		<!-- Quick Links -->
		<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
			<Card.Root class="hover:shadow-lg transition-shadow cursor-pointer">
				<a href="/dashboard/events" class="block">
					<Card.Header>
						<Card.Title class="text-lg">View Events</Card.Title>
						<Card.Description>Browse all security events</Card.Description>
					</Card.Header>
				</a>
			</Card.Root>

			<Card.Root class="hover:shadow-lg transition-shadow cursor-pointer">
				<a href="/dashboard/alerts" class="block">
					<Card.Header>
						<Card.Title class="text-lg">Manage Alerts</Card.Title>
						<Card.Description>Review and manage alerts</Card.Description>
					</Card.Header>
				</a>
			</Card.Root>

			<Card.Root class="hover:shadow-lg transition-shadow cursor-pointer">
				<a href="/dashboard/rules" class="block">
					<Card.Header>
						<Card.Title class="text-lg">Detection Rules</Card.Title>
						<Card.Description>Configure security rules</Card.Description>
					</Card.Header>
				</a>
			</Card.Root>

			<Card.Root class="hover:shadow-lg transition-shadow cursor-pointer">
				<a href="/dashboard/agents" class="block">
					<Card.Header>
						<Card.Title class="text-lg">Agent Status</Card.Title>
						<Card.Description>Monitor connected agents</Card.Description>
					</Card.Header>
				</a>
			</Card.Root>
		</div>
	{/if}
</div>
