<script lang="ts">
	import { resolve } from '$app/paths'
	import { page } from '$app/stores'
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query'
	import { toast } from 'svelte-sonner'
	import { clientFetch, type ApiResponse, type ApiResponseWithMetadata } from '$lib/client'
	import { V1_BASE_URL, type TableParams } from '$lib/constant'
	import {
		parseTableParams,
		updateTableParams,
		updateSearchParams,
		buildQueryString,
	} from '$lib/utils/table-params'
	import type { Alert, AlertDetail, AlertStatus } from '$lib/types/api'
	import AlertsTable from '$lib/components/table/alerts/alerts-table.svelte'
	import Loading from '$lib/components/ui/custom/loading.svelte'
	import Pagination from '$lib/components/ui/custom/pagination.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import SearchIcon from '@lucide/svelte/icons/search'
	import CalendarIcon from '@lucide/svelte/icons/calendar'
	import * as Table from '$lib/components/ui/table/index.js'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import { realtimeQueryOptions } from '$lib/utils/realtime-query'
	import { buildInvestigationHref, toEventTimeline } from '$lib/utils/investigation'

	type AlertPageParams = TableParams & {
		severity: string
		status: string
		triggered_from: string
		triggered_to: string
	}

	const queryClient = useQueryClient()

	const tableParams = $derived(parseTableParams($page.url))
	const params = $derived(parseAlertParams($page.url, tableParams))
	let selectedAlertID = $state<string | null>(null)
	let detailDialogOpen = $state(false)
	let triggeredFromDate = $state('')
	let triggeredToDate = $state('')

	$effect(() => {
		if (!detailDialogOpen) {
			selectedAlertID = null
		}
	})

	$effect(() => {
		triggeredFromDate = toDateInputValue(params.triggered_from)
		triggeredToDate = toDateInputValue(params.triggered_to)
	})

	const alertsQuery = createQuery<ApiResponseWithMetadata<Alert[]>>(() => ({
		queryKey: ['alerts', params],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Alert[]>>(
				`${V1_BASE_URL}/alerts?${buildAlertQueryString(params)}`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const alerts = $derived(alertsQuery.data?.data.items ?? [])
	const metadata = $derived(alertsQuery.data?.data.metadata)
	const totalPages = $derived(metadata?.total_pages ?? 1)

	const detailQuery = createQuery<ApiResponse<AlertDetail>>(() => ({
		queryKey: ['alert-detail', selectedAlertID],
		enabled: !!selectedAlertID,
		queryFn: () => clientFetch<ApiResponse<AlertDetail>>(`${V1_BASE_URL}/alerts/${selectedAlertID}`, { method: 'GET' }),
		...realtimeQueryOptions(!!selectedAlertID),
	}))

	const updateStatusMutation = createMutation(() => ({
		mutationFn: ({ id, status }: { id: string; status: AlertStatus }) =>
			clientFetch(`${V1_BASE_URL}/alerts/${id}/status`, {
				method: 'PATCH',
				body: JSON.stringify({ status }),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['alerts'] })
			if (selectedAlertID) {
				queryClient.invalidateQueries({ queryKey: ['alert-detail', selectedAlertID] })
			}
			toast.success('Alert status updated')
		},
		onError: (error: Error) => {
			toast.error(`Failed to update alert status: ${error.message}`)
		},
	}))

	function onSortChange(sort: string, order: 'asc' | 'desc') {
		updateTableParams({ sort, order }, $page.url)
	}

	function gotoPage(nextPage: number) {
		updateTableParams({ page: nextPage }, $page.url)
	}

	function handleLimitChange(value: string | undefined) {
		if (!value) return
		updateTableParams({ limit: parseInt(value), page: 1 }, $page.url)
	}

	function viewAlert(id: string) {
		selectedAlertID = id
		detailDialogOpen = true
	}

	function updateStatus(id: string, status: AlertStatus) {
		updateStatusMutation.mutate({ id, status })
	}

	function parseAlertParams(url: URL, table: TableParams): AlertPageParams {
		return {
			...table,
			severity: url.searchParams.get('severity') ?? '',
			status: url.searchParams.get('status') ?? '',
			triggered_from: url.searchParams.get('triggered_from') ?? '',
			triggered_to: url.searchParams.get('triggered_to') ?? '',
		}
	}

	function updateAlertExtraParams(next: Partial<Pick<AlertPageParams, 'severity' | 'status' | 'triggered_from' | 'triggered_to'>>) {
		const resetPage =
			('severity' in next && next.severity !== params.severity) ||
			('status' in next && next.status !== params.status) ||
			('triggered_from' in next && next.triggered_from !== params.triggered_from) ||
			('triggered_to' in next && next.triggered_to !== params.triggered_to)

		updateSearchParams(
			{
				severity: next.severity,
				status: next.status,
				triggered_from: next.triggered_from,
				triggered_to: next.triggered_to,
			},
			$page.url,
			{ resetPage },
		)
	}

	function buildAlertQueryString(p: AlertPageParams): string {
		const baseQuery = buildQueryString(p)
		const extras = [
			p.severity ? `severity=${encodeURIComponent(p.severity)}` : '',
			p.status ? `status=${encodeURIComponent(p.status)}` : '',
			p.triggered_from ? `triggered_from=${encodeURIComponent(p.triggered_from)}` : '',
			p.triggered_to ? `triggered_to=${encodeURIComponent(p.triggered_to)}` : '',
		]
			.filter(Boolean)
			.join('&')
		return extras ? `${baseQuery}&${extras}` : baseQuery
	}

	function toDateInputValue(value: string): string {
		if (!value) return ''
		return value.length >= 10 ? value.slice(0, 10) : ''
	}

	function fromDateInput(value: string, boundary: 'start' | 'end'): string {
		if (!value) return ''
		return boundary === 'start' ? `${value}T00:00:00.000Z` : `${value}T23:59:59.999Z`
	}

	function applyDateRange() {
		updateAlertExtraParams({
			triggered_from: fromDateInput(triggeredFromDate, 'start'),
			triggered_to: fromDateInput(triggeredToDate, 'end'),
		})
	}

	function clearDateRange() {
		triggeredFromDate = ''
		triggeredToDate = ''
		updateAlertExtraParams({ triggered_from: '', triggered_to: '' })
	}

	function buildEventsPivotHref(searchValue: string): string {
		return buildInvestigationHref(resolve('/management/events'), $page.url.origin, {
			search: searchValue,
		})
	}

	function buildEventsAgentPivotHref(agentID: string): string {
		return buildInvestigationHref(resolve('/management/events'), $page.url.origin, {
			agent_id: agentID,
		})
	}

	function buildEventsCategoryPivotHref(category: string): string {
		return buildInvestigationHref(resolve('/management/events'), $page.url.origin, {
			category,
		})
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Alerts</h1>
			<p class="text-muted-foreground">Monitor triggered detections and triage alert status</p>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<div class="relative flex-1 min-w-48 max-w-xs">
			<SearchIcon class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
			<Input
				class="pl-9"
				placeholder="Search alerts…"
				value={params.search}
				oninput={(e) => updateTableParams({ search: (e.target as HTMLInputElement).value }, $page.url)}
			/>
		</div>

		<Select.Root type="single" value={params.severity} onValueChange={(v) => updateAlertExtraParams({ severity: String(v ?? '') })}>
			<Select.Trigger class="w-[170px]">{params.severity || 'All severities'}</Select.Trigger>
			<Select.Content>
				<Select.Item value="">All severities</Select.Item>
				<Select.Item value="INFO">INFO</Select.Item>
				<Select.Item value="LOW">LOW</Select.Item>
				<Select.Item value="MEDIUM">MEDIUM</Select.Item>
				<Select.Item value="HIGH">HIGH</Select.Item>
				<Select.Item value="CRITICAL">CRITICAL</Select.Item>
			</Select.Content>
		</Select.Root>

		<Select.Root type="single" value={params.status} onValueChange={(v) => updateAlertExtraParams({ status: String(v ?? '') })}>
			<Select.Trigger class="w-[170px]">{params.status || 'All status'}</Select.Trigger>
			<Select.Content>
				<Select.Item value="">All status</Select.Item>
				<Select.Item value="new">new</Select.Item>
				<Select.Item value="acknowledged">acknowledged</Select.Item>
				<Select.Item value="closed">closed</Select.Item>
			</Select.Content>
		</Select.Root>

		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Button variant="outline" size="sm" {...props}>
						<CalendarIcon class="h-4 w-4 mr-2" />
						Date range
					</Button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align="end" class="w-[320px] p-3 space-y-3">
				<div class="space-y-1">
					<p class="text-xs text-muted-foreground">Date from</p>
					<Input type="date" bind:value={triggeredFromDate} />
				</div>
				<div class="space-y-1">
					<p class="text-xs text-muted-foreground">Date to</p>
					<Input type="date" bind:value={triggeredToDate} />
				</div>
				<div class="flex items-center justify-end gap-2">
					<Button variant="ghost" size="sm" onclick={clearDateRange}>Clear</Button>
					<Button variant="default" size="sm" onclick={applyDateRange}>Apply</Button>
				</div>
			</DropdownMenu.Content>
		</DropdownMenu.Root>

		{#if metadata}
			<span class="ml-auto text-sm text-muted-foreground">{metadata.total} alert{metadata.total !== 1 ? 's' : ''} total</span>
		{/if}
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		{#if alertsQuery.isPending}
			<Loading label="Loading alerts…" />
		{:else if alertsQuery.isError}
			<div class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
				<span class="text-destructive font-medium">Failed to load alerts</span>
				<span>{alertsQuery.error?.message}</span>
				<Button variant="outline" size="sm" onclick={() => alertsQuery.refetch()}>Try again</Button>
			</div>
		{:else if alerts.length === 0}
			<div class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
				<span>No alerts found</span>
			</div>
		{:else}
			<AlertsTable data={alerts} params={params} onSortChange={onSortChange} onView={viewAlert} onStatus={updateStatus} />
		{/if}
	</div>

	<div class="flex items-center justify-between">
		<LimitSelect value={params.limit} onValueChange={(v) => handleLimitChange(String(v))} />
		<Pagination page={params.page} {totalPages} onPageChange={gotoPage} />
	</div>

	<Dialog.Root bind:open={detailDialogOpen}>
		<Dialog.Content class="max-w-5xl">
			<Dialog.Header>
				<Dialog.Title>Alert Events</Dialog.Title>
				<Dialog.Description>Related events that triggered the selected alert.</Dialog.Description>
			</Dialog.Header>

			{#if selectedAlertID}
				{#if detailQuery.isPending}
					<Loading label="Loading alert events…" />
				{:else if detailQuery.isError}
					<div class="p-4 text-sm text-destructive">Failed to load alert events: {detailQuery.error?.message}</div>
				{:else if (detailQuery.data?.data.events?.length ?? 0) === 0}
					<div class="p-4 text-sm text-muted-foreground">No related events found for this alert.</div>
				{:else}
					{@const timeline = toEventTimeline(detailQuery.data?.data.events ?? [])}
					<div class="mb-3 rounded-md border p-3">
						<p class="mb-2 text-sm text-muted-foreground">Investigation timeline</p>
						<div class="max-h-40 overflow-auto rounded-md border bg-muted/20 p-3">
							<ul class="space-y-3 border-l pl-4">
								{#each timeline as item (item.id + (item.event_time || item.received_at || ''))}
									<li class="relative text-xs">
										<span class="absolute -left-[1.05rem] top-1 h-2 w-2 rounded-full bg-primary"></span>
										<p class="text-muted-foreground">
											{item.event_time || item.received_at}
											{#if item.severity} · {item.severity}{/if}
											{#if item.category} · {item.category}{/if}
										</p>
										<p class="break-words">{item.message}</p>
									</li>
								{/each}
							</ul>
						</div>
					</div>
					<div class="max-h-[60vh] overflow-auto rounded-md border">
						<Table.Root>
							<Table.Header>
								<Table.Row>
									<Table.Head>Event Time</Table.Head>
									<Table.Head>Hostname</Table.Head>
									<Table.Head>Source IP</Table.Head>
									<Table.Head>Category</Table.Head>
									<Table.Head>Severity</Table.Head>
									<Table.Head>Message</Table.Head>
								</Table.Row>
							</Table.Header>
							<Table.Body>
								{#each detailQuery.data?.data.events ?? [] as event (event.id + event.received_at)}
									<Table.Row>
										<Table.Cell><CompactDate dateString={event.event_time || event.received_at} /></Table.Cell>
										<Table.Cell>{event.hostname || '-'}</Table.Cell>
										<Table.Cell class="font-mono text-xs">
											{#if event.source_ip}
												<a class="underline underline-offset-2 hover:text-foreground/80" href={buildEventsPivotHref(event.source_ip)}>
													{event.source_ip}
												</a>
											{:else}
												-
											{/if}
										</Table.Cell>
										<Table.Cell>
											{#if event.category}
												<a class="underline underline-offset-2 hover:text-foreground/80" href={buildEventsCategoryPivotHref(event.category)}>
													{event.category}
												</a>
											{:else}
												-
											{/if}
										</Table.Cell>
										<Table.Cell>{event.severity || '-'}</Table.Cell>
										<Table.Cell class="max-w-[480px] truncate" title={event.message}>
											{event.message}
											{#if event.agent_id}
												<a class="ml-2 text-xs underline underline-offset-2 hover:text-foreground/80" href={buildEventsAgentPivotHref(event.agent_id)}>
													agent:{event.agent_id}
												</a>
											{/if}
										</Table.Cell>
									</Table.Row>
								{/each}
							</Table.Body>
						</Table.Root>
					</div>
				{/if}
			{/if}
		</Dialog.Content>
	</Dialog.Root>
</div>

<svelte:head>
	<title>Xemarify - Alerts</title>
</svelte:head>
