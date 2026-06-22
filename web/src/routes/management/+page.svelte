<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query'
	import {
		clientFetch,
		type ApiResponse,
		type ApiResponseWithCursorMetadata,
	} from '$lib/client'
	import { V1_BASE_URL } from '$lib/constant'
	import {
		REALTIME_INTERVAL_MS,
		realtimeQueryOptions,
	} from '$lib/utils/realtime-query'
	import type { Agent, Alert, DashboardStats } from '$lib/types/api'
	import { Button } from '$lib/components/ui/button/index.js'
	import * as Card from '$lib/components/ui/card/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'
	import * as Table from '$lib/components/ui/table/index.js'
	import * as Chart from '$lib/components/ui/chart/index.js'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import { AreaChart, BarChart } from 'layerchart'
	import FileTextIcon from '@lucide/svelte/icons/file-text'
	import BellIcon from '@lucide/svelte/icons/bell'
	import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert'
	import ServerIcon from '@lucide/svelte/icons/server'
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw'

	type DashboardMetric = {
		label: string
		value: string
		description: string
	}

	const trendChartConfig = {
		events: {
			label: 'Events',
			color: 'var(--chart-1)',
		},
		alerts: {
			label: 'Alerts',
			color: 'var(--chart-2)',
		},
	} satisfies Chart.ChartConfig

	const statusChartConfig = {
		count: {
			label: 'Alerts',
			color: 'var(--chart-3)',
		},
	} satisfies Chart.ChartConfig

	// Single stats query replaces all the individual summary queries
	const statsQuery = createQuery<ApiResponse<DashboardStats>>(() => ({
		queryKey: ['dashboard', 'stats'],
		queryFn: () =>
			clientFetch<ApiResponse<DashboardStats>>(
				`${V1_BASE_URL}/stats`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	// Agents query for online/offline counts (not in stats yet)
	const agentsQuery = createQuery<ApiResponseWithCursorMetadata<Agent[]>>(() => ({
		queryKey: ['dashboard', 'agents-overview'],
		queryFn: () =>
			clientFetch<ApiResponseWithCursorMetadata<Agent[]>>(
				`${V1_BASE_URL}/agents?limit=1000&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	// Recent critical alerts for the table at the bottom
	const recentAlertsQuery = createQuery<ApiResponseWithCursorMetadata<Alert[]>>(() => ({
		queryKey: ['dashboard', 'recent-critical-alerts'],
		queryFn: () =>
			clientFetch<ApiResponseWithCursorMetadata<Alert[]>>(
				`${V1_BASE_URL}/alerts?limit=20&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	let isRefreshing = $state(false)

	const summary = $derived(statsQuery.data?.data.summary)
	const trendData = $derived(
		(statsQuery.data?.data.activity_trend ?? []).map((p) => ({
			...p,
			day: formatDayLabel(p.day),
		})),
	)
	const statusData = $derived(
		buildStatusData(statsQuery.data?.data.alert_status_distribution ?? []),
	)

	const agents = $derived(agentsQuery.data?.data.items ?? [])
	const totalAgents = $derived(summary?.total_agents ?? agents.length)
	const onlineAgents = $derived(
		summary?.online_agents ??
		agents.filter((a) => a.status === 'ONLINE').length
	)
	const agentCoverage = $derived(
		totalAgents > 0 ? Math.round((onlineAgents / totalAgents) * 100) : 0,
	)

	const recentAlertsSample = $derived(recentAlertsQuery.data?.data.items ?? [])
	const recentCriticalAlerts = $derived(
		recentAlertsSample
			.filter(
				(alert) => alert.severity === 'CRITICAL' || alert.severity === 'HIGH',
			)
			.slice(0, 5),
	)

	const metrics = $derived<DashboardMetric[]>([
		{
			label: 'Total Events',
			value: formatNumber(summary?.total_events ?? 0),
			description: 'Ingested events (last 30 days)',
		},
		{
			label: 'Total Alerts',
			value: formatNumber(summary?.total_alerts ?? 0),
			description: 'Detected alerts',
		},
		{
			label: 'New Alerts',
			value: formatNumber(summary?.new_alerts ?? 0),
			description: 'Needs triage',
		},
		{
			label: 'Agent Coverage',
			value: `${formatNumber(agentCoverage)}%`,
			description: `${formatNumber(onlineAgents)}/${formatNumber(totalAgents)} agents online`,
		},
	])

	const latestUpdateTimestamp = $derived(
		Math.max(
			statsQuery.dataUpdatedAt ?? 0,
			agentsQuery.dataUpdatedAt ?? 0,
			recentAlertsQuery.dataUpdatedAt ?? 0,
		),
	)

	async function refreshDashboard() {
		isRefreshing = true
		await Promise.allSettled([
			statsQuery.refetch(),
			agentsQuery.refetch(),
			recentAlertsQuery.refetch(),
		])
		isRefreshing = false
	}

	function formatNumber(value: number): string {
		return new Intl.NumberFormat('id-ID').format(value)
	}

	type StatusDistItem = { status: string; count: number }

	function buildStatusData(dist: StatusDistItem[]) {
		// Ensure all three statuses always appear in the chart
		const base = new Map<string, number>([
			['new', 0],
			['acknowledged', 0],
			['closed', 0],
		])
		for (const item of dist) {
			base.set(item.status, item.count)
		}
		const label: Record<string, string> = {
			new: 'New',
			acknowledged: 'Acknowledged',
			closed: 'Closed',
		}
		return Array.from(base.entries()).map(([status, count]) => ({
			status: label[status] ?? status,
			count,
		}))
	}

	function formatDayLabel(isoDate: string): string {
		const date = new Date(isoDate)
		return date.toLocaleDateString('id-ID', {
			weekday: 'short',
			day: '2-digit',
		})
	}

	function getAlertVariant(status: string):
		| 'default'
		| 'secondary'
		| 'destructive'
		| 'outline' {
		if (status === 'new') return 'destructive'
		if (status === 'acknowledged') return 'default'
		if (status === 'closed') return 'secondary'
		return 'outline'
	}

	function getSeverityVariant(severity: string):
		| 'default'
		| 'secondary'
		| 'destructive'
		| 'outline' {
		if (severity === 'CRITICAL') return 'destructive'
		if (severity === 'HIGH') return 'default'
		if (severity === 'MEDIUM') return 'secondary'
		return 'outline'
	}

	function formatLastUpdated(timestamp: number): string {
		if (!timestamp) return 'Waiting for telemetry'
		return new Date(timestamp).toLocaleTimeString('id-ID', {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
		})
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Dashboard</h1>
			<p class="text-muted-foreground">
				SIEM overview with automatic updates every {Math.round(
					REALTIME_INTERVAL_MS / 1000,
				)} seconds.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<div class="hidden rounded-lg border bg-card px-3 py-2 text-sm md:block">
				<p class="text-muted-foreground">Last updated</p>
				<p class="font-medium">{formatLastUpdated(latestUpdateTimestamp)}</p>
			</div>
			<Button variant="outline" onclick={refreshDashboard} disabled={isRefreshing}>
				<RefreshCwIcon class={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
				{isRefreshing ? 'Refreshing...' : 'Refresh'}
			</Button>
		</div>
	</div>

	<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
		<Card.Root>
			<Card.Content class="flex items-start justify-between px-5 py-5">
				<div>
					<p class="text-sm text-muted-foreground">{metrics[0].label}</p>
					<p class="mt-2 text-2xl font-semibold leading-none">{metrics[0].value}</p>
					<p class="mt-1.5 text-xs text-muted-foreground">{metrics[0].description}</p>
				</div>
				<FileTextIcon class="h-4 w-4 shrink-0 text-muted-foreground" />
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Content class="flex items-start justify-between px-5 py-5">
				<div>
					<p class="text-sm text-muted-foreground">{metrics[1].label}</p>
					<p class="mt-2 text-2xl font-semibold leading-none">{metrics[1].value}</p>
					<p class="mt-1.5 text-xs text-muted-foreground">{metrics[1].description}</p>
				</div>
				<BellIcon class="h-4 w-4 shrink-0 text-muted-foreground" />
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Content class="flex items-start justify-between px-5 py-5">
				<div>
					<p class="text-sm text-muted-foreground">{metrics[2].label}</p>
					<p class="mt-2 text-2xl font-semibold leading-none text-destructive">{metrics[2].value}</p>
					<p class="mt-1.5 text-xs text-muted-foreground">{metrics[2].description}</p>
				</div>
				<ShieldAlertIcon class="h-4 w-4 shrink-0 text-destructive" />
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Content class="flex items-start justify-between px-5 py-5">
				<div>
					<p class="text-sm text-muted-foreground">{metrics[3].label}</p>
					<p class="mt-2 text-2xl font-semibold leading-none">{metrics[3].value}</p>
					<p class="mt-1.5 text-xs text-muted-foreground">{metrics[3].description}</p>
				</div>
				<ServerIcon class="h-4 w-4 shrink-0 text-muted-foreground" />
			</Card.Content>
		</Card.Root>
	</div>

	<div class="grid gap-4 xl:grid-cols-[1.7fr_1fr]">
		<Card.Root>
			<Card.Header>
				<Card.Title>Activity Trend</Card.Title>
				<Card.Description>
					Event and alert activity over the last 7 days.
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<Chart.Container config={trendChartConfig} class="min-h-[260px] w-full">
					<AreaChart
						data={trendData}
						x="day"
						axis="x"
						legend
						series={[
							{
								key: 'events',
								label: trendChartConfig.events.label,
								color: trendChartConfig.events.color,
							},
							{
								key: 'alerts',
								label: trendChartConfig.alerts.label,
								color: trendChartConfig.alerts.color,
							},
						]}
					>
						{#snippet tooltip()}
							<Chart.Tooltip />
						{/snippet}
					</AreaChart>
				</Chart.Container>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Alert Status</Card.Title>
				<Card.Description>
					Status distribution across all alerts.
				</Card.Description>
			</Card.Header>
			<Card.Content class="space-y-4">
				<Chart.Container config={statusChartConfig} class="min-h-[260px] w-full">
					<BarChart
						data={statusData}
						x="status"
						y="count"
						axis="x"
						series={[
							{
								key: 'count',
								label: statusChartConfig.count.label,
								color: statusChartConfig.count.color,
							},
						]}
					>
						{#snippet tooltip()}
							<Chart.Tooltip />
						{/snippet}
					</BarChart>
				</Chart.Container>

				<div class="grid grid-cols-3 gap-2 text-sm">
					<div class="rounded-lg bg-muted/50 p-3">
						<p class="text-muted-foreground">Audit Logs</p>
						<p class="mt-1 font-semibold">{formatNumber(summary?.audit_log_total ?? 0)}</p>
					</div>
					<div class="rounded-lg bg-muted/50 p-3">
						<p class="text-muted-foreground">Total Events</p>
						<p class="mt-1 font-semibold">{formatNumber(summary?.total_events ?? 0)}</p>
					</div>
					<div class="rounded-lg bg-muted/50 p-3">
						<p class="text-muted-foreground">Total Alerts</p>
						<p class="mt-1 font-semibold">{formatNumber(summary?.total_alerts ?? 0)}</p>
					</div>
				</div>
			</Card.Content>
		</Card.Root>
	</div>

	<Card.Root>
		<Card.Header>
			<div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
				<div>
					<Card.Title>Recent Critical Alerts</Card.Title>
					<Card.Description>
						A short list of high-priority alerts for quick escalation.
					</Card.Description>
				</div>
				<Button variant="outline" href="/management/alerts">Open alerts</Button>
			</div>
		</Card.Header>
		<Card.Content>
			{#if recentCriticalAlerts.length === 0}
				<div class="rounded-lg border border-dashed px-4 py-10 text-center text-sm text-muted-foreground">
					No `HIGH/CRITICAL` alerts found in recent data.
				</div>
			{:else}
				<div class="overflow-hidden rounded-lg border">
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head>Rule</Table.Head>
								<Table.Head>Severity</Table.Head>
								<Table.Head>Status</Table.Head>
								<Table.Head>Triggered</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each recentCriticalAlerts as alert (alert.id)}
								<Table.Row>
									<Table.Cell>
										<div class="space-y-1">
											<p class="font-medium">{alert.rule_name}</p>
											<p class="text-xs text-muted-foreground">
												Correlation key: {alert.correlation_key}
											</p>
										</div>
									</Table.Cell>
									<Table.Cell>
										<Badge variant={getSeverityVariant(alert.severity)}>{alert.severity}</Badge>
									</Table.Cell>
									<Table.Cell>
										<Badge variant={getAlertVariant(alert.status)}>{alert.status}</Badge>
									</Table.Cell>
									<Table.Cell>
										<CompactDate dateString={alert.triggered_at} />
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				</div>
			{/if}
		</Card.Content>
	</Card.Root>
</div>

<svelte:head>
	<title>Xemarify - Dashboard</title>
</svelte:head>
