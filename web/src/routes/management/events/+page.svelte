<script lang="ts">
	import { page } from '$app/stores'
	import { createQuery } from '@tanstack/svelte-query'
	import { clientFetch, type ApiResponse, type ApiResponseWithMetadata } from '$lib/client'
	import { V1_BASE_URL, type TableParams } from '$lib/constant'
	import {
		parseTableParams,
		updateTableParams,
		updateSearchParams,
	} from '$lib/utils/table-params'
	import type { EventDetail, EventItem } from '$lib/types/api'
	import Loading from '$lib/components/ui/custom/loading.svelte'
	import Pagination from '$lib/components/ui/custom/pagination.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'
	import EventsDataTable from '$lib/components/table/events/events-table.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import SearchIcon from '@lucide/svelte/icons/search'
	import CalendarIcon from '@lucide/svelte/icons/calendar'

	type EventPageParams = TableParams & {
		severity: string
		category: string
		agent_id: string
		date_from: string
		date_to: string
	}

	const tableParams = $derived(parseTableParams($page.url))
	const params = $derived(parseEventParams($page.url, tableParams))
	let dateFrom = $state('')
	let dateTo = $state('')
	let selectedEventID = $state<string | null>(null)
	let detailDialogOpen = $state(false)

	$effect(() => {
		if (!detailDialogOpen) {
			selectedEventID = null
		}
	})

	$effect(() => {
		dateFrom = toDateInputValue(params.date_from)
		dateTo = toDateInputValue(params.date_to)
	})

	const eventsQuery = createQuery<ApiResponseWithMetadata<EventItem[]>>(() => ({
		queryKey: ['events', params],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<EventItem[]>>(
				`${V1_BASE_URL}/events?${buildEventsQueryString(params)}`,
				{ method: 'GET' },
			),
	}))

	const events = $derived(eventsQuery.data?.data.items ?? [])
	const metadata = $derived(eventsQuery.data?.data.metadata)
	const totalPages = $derived(metadata?.total_pages ?? 1)

	const detailQuery = createQuery<ApiResponse<EventDetail>>(() => ({
		queryKey: ['event-detail', selectedEventID],
		enabled: !!selectedEventID,
		queryFn: () => clientFetch<ApiResponse<EventDetail>>(`${V1_BASE_URL}/events/${selectedEventID}`, { method: 'GET' }),
	}))

	function parseEventParams(url: URL, table: TableParams): EventPageParams {
		return {
			...table,
			severity: url.searchParams.get('severity') ?? '',
			category: url.searchParams.get('category') ?? '',
			agent_id: url.searchParams.get('agent_id') ?? '',
			date_from: url.searchParams.get('date_from') ?? '',
			date_to: url.searchParams.get('date_to') ?? '',
		}
	}

	function buildEventsQueryString(p: EventPageParams): string {
		const qs = new URLSearchParams()
		qs.set('limit', String(p.limit))
		qs.set('offset', String(Math.max(0, (p.page - 1) * p.limit)))
		qs.set('sort_by', p.sort)
		qs.set('order', p.order)
		if (p.search) qs.set('search', p.search)
		if (p.severity) qs.set('severity', p.severity)
		if (p.category) qs.set('category', p.category)
		if (p.agent_id) qs.set('agent_id', p.agent_id)
		if (p.date_from) qs.set('date_from', p.date_from)
		if (p.date_to) qs.set('date_to', p.date_to)
		return qs.toString()
	}

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

	function updateExtraParams(next: Partial<Pick<EventPageParams, 'severity' | 'category' | 'agent_id' | 'date_from' | 'date_to'>>) {
		const resetPage =
			('severity' in next && next.severity !== params.severity) ||
			('category' in next && next.category !== params.category) ||
			('agent_id' in next && next.agent_id !== params.agent_id) ||
			('date_from' in next && next.date_from !== params.date_from) ||
			('date_to' in next && next.date_to !== params.date_to)

		updateSearchParams(
			{
				severity: next.severity,
				category: next.category,
				agent_id: next.agent_id,
				date_from: next.date_from,
				date_to: next.date_to,
			},
			$page.url,
			{ resetPage },
		)
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
		updateExtraParams({
			date_from: fromDateInput(dateFrom, 'start'),
			date_to: fromDateInput(dateTo, 'end'),
		})
	}

	function clearDateRange() {
		dateFrom = ''
		dateTo = ''
		updateExtraParams({ date_from: '', date_to: '' })
	}

	function viewEvent(id: string) {
		selectedEventID = id
		detailDialogOpen = true
	}

	function stringifyNormalized(normalized: Record<string, unknown> | undefined): string {
		if (!normalized || Object.keys(normalized).length === 0) {
			return '-'
		}
		return JSON.stringify(normalized, null, 2)
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Events</h1>
			<p class="text-muted-foreground">Inspect ingested security events from agents</p>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<div class="relative flex-1 min-w-48 max-w-xs">
			<SearchIcon class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
			<Input
				class="pl-9"
				placeholder="Search events…"
				value={params.search}
				oninput={(e) => updateTableParams({ search: (e.target as HTMLInputElement).value }, $page.url)}
			/>
		</div>

		<Select.Root type="single" value={params.severity} onValueChange={(v) => updateExtraParams({ severity: String(v ?? '') })}>
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

		<Input
			class="w-[170px]"
			placeholder="Category"
			value={params.category}
			onchange={(e) => updateExtraParams({ category: (e.target as HTMLInputElement).value })}
		/>

		<Input
			class="w-[220px]"
			placeholder="Agent ID"
			value={params.agent_id}
			onchange={(e) => updateExtraParams({ agent_id: (e.target as HTMLInputElement).value })}
		/>

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
					<Input type="date" bind:value={dateFrom} />
				</div>
				<div class="space-y-1">
					<p class="text-xs text-muted-foreground">Date to</p>
					<Input type="date" bind:value={dateTo} />
				</div>
				<div class="flex items-center justify-end gap-2">
					<Button variant="ghost" size="sm" onclick={clearDateRange}>Clear</Button>
					<Button variant="default" size="sm" onclick={applyDateRange}>Apply</Button>
				</div>
			</DropdownMenu.Content>
		</DropdownMenu.Root>

		{#if metadata}
			<span class="ml-auto text-sm text-muted-foreground">{metadata.total} event{metadata.total !== 1 ? 's' : ''} total</span>
		{/if}
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		{#if eventsQuery.isPending}
			<Loading label="Loading events…" />
		{:else if eventsQuery.isError}
			<div class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
				<span class="text-destructive font-medium">Failed to load events</span>
				<span>{eventsQuery.error?.message}</span>
				<Button variant="outline" size="sm" onclick={() => eventsQuery.refetch()}>Try again</Button>
			</div>
		{:else if events.length === 0}
			<div class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
				<span>No events found</span>
			</div>
		{:else}
			<EventsDataTable data={events} {params} {onSortChange} onView={viewEvent} />
		{/if}
	</div>

	<div class="flex items-center justify-between">
		<LimitSelect value={params.limit} onValueChange={(v) => handleLimitChange(String(v))} />
		<Pagination page={params.page} {totalPages} onPageChange={gotoPage} />
	</div>

	<Dialog.Root bind:open={detailDialogOpen}>
		<Dialog.Content class="max-w-4xl">
			<Dialog.Header>
				<Dialog.Title>Event Details</Dialog.Title>
				<Dialog.Description>Inspect complete payload for the selected event.</Dialog.Description>
			</Dialog.Header>

			{#if selectedEventID}
				{#if detailQuery.isPending}
					<Loading label="Loading event details…" />
				{:else if detailQuery.isError}
					<div class="p-4 text-sm text-destructive">Failed to load event details: {detailQuery.error?.message}</div>
				{:else if !detailQuery.data?.data}
					<div class="p-4 text-sm text-muted-foreground">Event detail not found.</div>
				{:else}
					{@const detail = detailQuery.data.data}
					<div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
						<div class="rounded-md border p-3 space-y-2">
							<p><span class="text-muted-foreground">Event ID:</span> <span class="font-mono text-xs">{detail.id}</span></p>
							<p><span class="text-muted-foreground">Agent ID:</span> <span class="font-mono text-xs">{detail.agent_id}</span></p>
							<p><span class="text-muted-foreground">Hostname:</span> {detail.hostname || '-'}</p>
							<p><span class="text-muted-foreground">Source IP:</span> {detail.source_ip || '-'}</p>
							<p><span class="text-muted-foreground">Severity:</span> {detail.severity || '-'}</p>
							<p><span class="text-muted-foreground">Category:</span> {detail.category || '-'}</p>
						</div>
						<div class="rounded-md border p-3 space-y-2">
							<p><span class="text-muted-foreground">Event Time:</span> {detail.event_time}</p>
							<p><span class="text-muted-foreground">Received At:</span> {detail.received_at}</p>
							<p><span class="text-muted-foreground">Input Type:</span> {detail.input_type || '-'}</p>
							<p><span class="text-muted-foreground">Facility:</span> {detail.facility || '-'}</p>
							<p><span class="text-muted-foreground">Message:</span></p>
							<p class="rounded bg-muted/50 px-2 py-1 break-words">{detail.message}</p>
						</div>
					</div>
					<div class="mt-3 rounded-md border p-3">
						<p class="text-sm text-muted-foreground mb-2">Normalized</p>
						<pre class="text-xs max-h-48 overflow-auto rounded bg-muted/50 p-2 whitespace-pre-wrap break-words">{stringifyNormalized(detail.normalized)}</pre>
					</div>
					<div class="mt-3 rounded-md border p-3">
						<p class="text-sm text-muted-foreground mb-2">Raw</p>
						<pre class="text-xs max-h-48 overflow-auto rounded bg-muted/50 p-2 whitespace-pre-wrap break-words">{detail.raw || '-'}</pre>
					</div>
				{/if}
			{/if}
		</Dialog.Content>
	</Dialog.Root>
</div>
