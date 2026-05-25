<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query'
	import { clientFetch, type ApiResponseWithMetadata } from '$lib/client'
	import { V1_BASE_URL } from '$lib/constant'
	import {
		REALTIME_INTERVAL_MS,
		realtimeQueryOptions,
	} from '$lib/utils/realtime-query'
	import type { Agent, Alert, AuditLog, EventItem } from '$lib/types/api'
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

	type TrendDatum = {
		day: string
		events: number
		alerts: number
	}

	type StatusDatum = {
		status: string
		count: number
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

	const eventsSummaryQuery = createQuery<ApiResponseWithMetadata<EventItem[]>>(() => ({
		queryKey: ['dashboard', 'events-summary'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<EventItem[]>>(
				`${V1_BASE_URL}/events?limit=1&offset=0&sort_by=received_at&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const alertsSummaryQuery = createQuery<ApiResponseWithMetadata<Alert[]>>(() => ({
		queryKey: ['dashboard', 'alerts-summary'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Alert[]>>(
				`${V1_BASE_URL}/alerts?limit=1&offset=0&sort_by=triggered_at&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const newAlertsSummaryQuery = createQuery<ApiResponseWithMetadata<Alert[]>>(() => ({
		queryKey: ['dashboard', 'alerts-summary', 'new'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Alert[]>>(
				`${V1_BASE_URL}/alerts?limit=1&offset=0&sort_by=triggered_at&order=desc&status=new`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const auditLogsSummaryQuery = createQuery<ApiResponseWithMetadata<AuditLog[]>>(() => ({
		queryKey: ['dashboard', 'audit-logs-summary'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<AuditLog[]>>(
				`${V1_BASE_URL}/audit-logs?limit=1&offset=0&sort_by=created_at&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const agentsQuery = createQuery<ApiResponseWithMetadata<Agent[]>>(() => ({
		queryKey: ['dashboard', 'agents-overview'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Agent[]>>(
				`${V1_BASE_URL}/agents?limit=100&offset=0&sort_by=created_at&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const eventsSampleQuery = createQuery<ApiResponseWithMetadata<EventItem[]>>(() => ({
		queryKey: ['dashboard', 'events-sample'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<EventItem[]>>(
				`${V1_BASE_URL}/events?limit=80&offset=0&sort_by=received_at&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const alertsSampleQuery = createQuery<ApiResponseWithMetadata<Alert[]>>(() => ({
		queryKey: ['dashboard', 'alerts-sample'],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Alert[]>>(
				`${V1_BASE_URL}/alerts?limit=40&offset=0&sort_by=triggered_at&order=desc`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	let isRefreshing = $state(false)

	const eventsTotal = $derived(eventsSummaryQuery.data?.data.metadata.total ?? 0)
	const alertsTotal = $derived(alertsSummaryQuery.data?.data.metadata.total ?? 0)
	const newAlertsTotal = $derived(newAlertsSummaryQuery.data?.data.metadata.total ?? 0)
	const auditLogsTotal = $derived(auditLogsSummaryQuery.data?.data.metadata.total ?? 0)
	const agents = $derived(agentsQuery.data?.data.items ?? [])
	const totalAgents = $derived(agentsQuery.data?.data.metadata.total ?? agents.length)
	const onlineAgents = $derived(agents.filter((agent) => agent.status === 'ONLINE').length)
	const agentCoverage = $derived(
		totalAgents > 0 ? Math.round((onlineAgents / totalAgents) * 100) : 0,
	)
	const eventsSample = $derived(eventsSampleQuery.data?.data.items ?? [])
	const alertsSample = $derived(alertsSampleQuery.data?.data.items ?? [])
	const recentCriticalAlerts = $derived(
		alertsSample
			.filter((alert) => alert.severity === 'CRITICAL' || alert.severity === 'HIGH')
			.slice(0, 5),
	)
	const metrics = $derived<DashboardMetric[]>([
		{
			label: 'Total Events',
			value: formatNumber(eventsTotal),
			description: 'Ingested events',
		},
		{
			label: 'Total Alerts',
			value: formatNumber(alertsTotal),
			description: 'Detected alerts',
		},
		{
			label: 'New Alerts',
			value: formatNumber(newAlertsTotal),
			description: 'Needs triage',
		},
		{
			label: 'Agent Coverage',
			value: `${formatNumber(agentCoverage)}%`,
			description: `${formatNumber(onlineAgents)}/${formatNumber(totalAgents)} agents online`,
		},
	])
	const trendData = $derived(buildTrendData(eventsSample, alertsSample))
	const statusData = $derived(buildStatusData(alertsSample))
	const latestUpdateTimestamp = $derived(
		Math.max(
			eventsSummaryQuery.dataUpdatedAt ?? 0,
			alertsSummaryQuery.dataUpdatedAt ?? 0,
			newAlertsSummaryQuery.dataUpdatedAt ?? 0,
			auditLogsSummaryQuery.dataUpdatedAt ?? 0,
			agentsQuery.dataUpdatedAt ?? 0,
			eventsSampleQuery.dataUpdatedAt ?? 0,
			alertsSampleQuery.dataUpdatedAt ?? 0,
		),
	)

	async function refreshDashboard() {
		isRefreshing = true
		await Promise.allSettled([
			eventsSummaryQuery.refetch(),
			alertsSummaryQuery.refetch(),
			newAlertsSummaryQuery.refetch(),
			auditLogsSummaryQuery.refetch(),
			agentsQuery.refetch(),
			eventsSampleQuery.refetch(),
			alertsSampleQuery.refetch(),
		])
		isRefreshing = false
	}

	function formatNumber(value: number): string {
		return new Intl.NumberFormat('id-ID').format(value)
	}

	function buildTrendData(events: EventItem[], alerts: Alert[]): TrendDatum[] {
		const buckets = new Map<string, TrendDatum>()

		for (let index = 6; index >= 0; index -= 1) {
			const day = new Date()
			day.setHours(0, 0, 0, 0)
			day.setDate(day.getDate() - index)
			const key = toDayKey(day)
			buckets.set(key, {
				day: formatDayLabel(day),
				events: 0,
				alerts: 0,
			})
		}

		for (const event of events) {
			const key = toDayKey(new Date(event.received_at || event.event_time))
			const bucket = buckets.get(key)
			if (bucket) bucket.events += 1
		}

		for (const alert of alerts) {
			const key = toDayKey(new Date(alert.triggered_at))
			const bucket = buckets.get(key)
			if (bucket) bucket.alerts += 1
		}

		return Array.from(buckets.values())
	}

	function buildStatusData(alerts: Alert[]): StatusDatum[] {
		const counts = new Map<string, number>([
			['New', 0],
			['Acknowledged', 0],
			['Closed', 0],
		])

		for (const alert of alerts) {
			if (alert.status === 'new') counts.set('New', (counts.get('New') ?? 0) + 1)
			else if (alert.status === 'acknowledged') {
				counts.set('Acknowledged', (counts.get('Acknowledged') ?? 0) + 1)
			} else if (alert.status === 'closed') {
				counts.set('Closed', (counts.get('Closed') ?? 0) + 1)
			}
		}

		return Array.from(counts.entries()).map(([status, count]) => ({ status, count }))
	}

	function toDayKey(date: Date): string {
		const year = date.getFullYear()
		const month = String(date.getMonth() + 1).padStart(2, '0')
		const day = String(date.getDate()).padStart(2, '0')
		return `${year}-${month}-${day}`
	}

	function formatDayLabel(date: Date): string {
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
					Event and alert activity over the last 7 days based on the latest data.
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
					Status distribution from the latest alert sample.
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
						<p class="mt-1 font-semibold">{formatNumber(auditLogsTotal)}</p>
					</div>
					<div class="rounded-lg bg-muted/50 p-3">
						<p class="text-muted-foreground">Sample Events</p>
						<p class="mt-1 font-semibold">{formatNumber(eventsSample.length)}</p>
					</div>
					<div class="rounded-lg bg-muted/50 p-3">
						<p class="text-muted-foreground">Sample Alerts</p>
						<p class="mt-1 font-semibold">{formatNumber(alertsSample.length)}</p>
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
					No `HIGH/CRITICAL` alerts found in the latest sample.
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
