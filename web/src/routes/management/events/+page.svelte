<script lang="ts">
	import { resolve } from '$app/paths'
	import { page } from '$app/stores'
	import { createQuery, useQueryClient } from '@tanstack/svelte-query'
	import {
		clientFetch,
		type ApiResponse,
		type ApiResponseWithCursorMetadata,
	} from '$lib/client'
	import { V1_BASE_URL, type TableParams } from '$lib/constant'
	import {
		parseTableParams,
		updateTableParams,
		updateSearchParams,
	} from '$lib/utils/table-params'
	import type { EventDetail, EventItem } from '$lib/types/api'
	import Loading from '$lib/components/ui/custom/loading.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'
	import EventsDataTable from '$lib/components/table/events/events-table.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import * as Tabs from '$lib/components/ui/tabs/index.js'
	import SearchIcon from '@lucide/svelte/icons/search'
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left'
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right'
	import CalendarIcon from '@lucide/svelte/icons/calendar'
	import FingerprintIcon from '@lucide/svelte/icons/fingerprint'
	import ClockIcon from '@lucide/svelte/icons/clock'
	import BracesIcon from '@lucide/svelte/icons/braces'
	import ActivityIcon from '@lucide/svelte/icons/activity'
	import { createEventSource } from '$lib/utils/event-source'
	import {
		buildInvestigationHref,
		toEventTimeline,
	} from '$lib/utils/investigation'
	import { formatDateTime, formatDateTimeExact } from '$lib/utils/date'

	type EventPageParams = Omit<TableParams, 'page'> & {
		severity: string
		category: string
		agent_id: string
		date_from: string
		date_to: string
	}

	const queryClient = useQueryClient()
	const tableParams = $derived(parseTableParams($page.url))
	const params = $derived(parseEventParams($page.url, tableParams))
	// cursorStack[0] = '' means first page; each subsequent entry is the next_cursor
	// from the previous page response.
	let cursorStack = $state<string[]>([''])
	const currentCursor = $derived(cursorStack[cursorStack.length - 1])
	const currentPage = $derived(cursorStack.length) // 1-indexed display

	const filterKey = $derived(
		[
			params.search,
			params.severity,
			params.category,
			params.agent_id,
			params.date_from,
			params.date_to,
			params.limit,
			params.order,
		].join('|'),
	)
	let prevFilterKey = $state('')
	$effect(() => {
		if (filterKey !== prevFilterKey) {
			prevFilterKey = filterKey
			cursorStack = ['']
		}
	})

	let dateFrom = $state('')
	let dateTo = $state('')
	let selectedEventID = $state<string | null>(null)
	let detailDialogOpen = $state(false)
	let sseStatus = $state<'connected' | 'connecting' | 'disconnected'>(
		'disconnected',
	)

	$effect(() => {
		if (!detailDialogOpen) {
			selectedEventID = null
		}
	})

	$effect(() => {
		dateFrom = toDateInputValue(params.date_from)
		dateTo = toDateInputValue(params.date_to)
	})

	const eventsQuery = createQuery<ApiResponseWithCursorMetadata<EventItem[]>>(
		() => ({
			queryKey: ['events', params, currentCursor],
			queryFn: () =>
				clientFetch<ApiResponseWithCursorMetadata<EventItem[]>>(
					`${V1_BASE_URL}/events?${buildEventsQueryString(params, currentCursor)}`,
					{ method: 'GET' },
				),
			refetchOnWindowFocus: true,
		}),
	)

	// SSE connection for realtime event updates.
	let cleanupSSE: (() => void) | undefined

	$effect(() => {
		cleanupSSE = createEventSource({
			onEvent: {
				new_event: (_event: MessageEvent) => {
					queryClient.invalidateQueries({ queryKey: ['events'] })
				},
			},
			onStatus: (status) => {
				sseStatus = status
			},
		})

		return () => {
			cleanupSSE?.()
		}
	})

	const events = $derived(eventsQuery.data?.data.items ?? [])
	const metadata = $derived(eventsQuery.data?.data.metadata)
	const hasMore = $derived(metadata?.has_more ?? false)

	const detailQuery = createQuery<ApiResponse<EventDetail>>(() => ({
		queryKey: ['event-detail', selectedEventID],
		enabled: !!selectedEventID,
		queryFn: () =>
			clientFetch<ApiResponse<EventDetail>>(
				`${V1_BASE_URL}/events/${selectedEventID}`,
				{ method: 'GET' },
			),
		refetchOnWindowFocus: true,
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

	function buildEventsQueryString(p: EventPageParams, cursor: string): string {
		const qs = new URLSearchParams()
		qs.set('limit', String(p.limit))
		qs.set('order', p.order)
		if (cursor) qs.set('cursor', cursor)
		if (p.search) qs.set('search', p.search)
		if (p.severity) qs.set('severity', p.severity)
		if (p.category) qs.set('category', p.category)
		if (p.agent_id) qs.set('agent_id', p.agent_id)
		if (p.date_from) qs.set('date_from', p.date_from)
		if (p.date_to) qs.set('date_to', p.date_to)
		return qs.toString()
	}

	function goNext() {
		const nc = metadata?.next_cursor
		if (!nc) return
		cursorStack = [...cursorStack, nc]
	}

	function goPrev() {
		if (cursorStack.length <= 1) return
		cursorStack = cursorStack.slice(0, -1)
	}

	function handleLimitChange(value: string | undefined) {
		if (!value) return
		updateTableParams({ limit: parseInt(value) }, $page.url)
	}

	function updateExtraParams(
		next: Partial<
			Pick<
				EventPageParams,
				'severity' | 'category' | 'agent_id' | 'date_from' | 'date_to'
			>
		>,
	) {
		updateSearchParams(
			{
				severity: next.severity,
				category: next.category,
				agent_id: next.agent_id,
				date_from: next.date_from,
				date_to: next.date_to,
			},
			$page.url,
		)
	}

	function toDateInputValue(value: string): string {
		if (!value) return ''
		return value.length >= 10 ? value.slice(0, 10) : ''
	}

	function fromDateInput(value: string, boundary: 'start' | 'end'): string {
		if (!value) return ''
		return boundary === 'start'
			? `${value}T00:00:00.000Z`
			: `${value}T23:59:59.999Z`
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

	function stringifyNormalized(
		normalized: Record<string, unknown> | undefined,
	): string {
		if (!normalized || Object.keys(normalized).length === 0) {
			return '-'
		}
		return JSON.stringify(normalized, null, 2)
	}

	function buildAlertsPivotHref(searchValue: string): string {
		return buildInvestigationHref(
			resolve('/management/alerts'),
			$page.url.origin,
			{
				search: searchValue,
			},
		)
	}

	function buildEventsSearchPivotHref(searchValue: string): string {
		return buildInvestigationHref(
			resolve('/management/events'),
			$page.url.origin,
			{
				search: searchValue,
			},
		)
	}

	function buildEventsFilterPivotHref(
		next: Partial<Pick<EventPageParams, 'agent_id' | 'category'>>,
	) {
		return buildInvestigationHref(
			resolve('/management/events'),
			$page.url.origin,
			{
				agent_id: next.agent_id,
				category: next.category,
			},
		)
	}

	function severityClass(severity: string): string {
		const map: Record<string, string> = {
			CRITICAL: 'bg-red-600 hover:bg-red-600 text-white border-transparent',
			HIGH: 'bg-orange-500 hover:bg-orange-500 text-white border-transparent',
			MEDIUM: 'bg-yellow-500 hover:bg-yellow-500 text-white border-transparent',
			LOW: 'bg-blue-500 hover:bg-blue-500 text-white border-transparent',
			INFO: 'bg-slate-500 hover:bg-slate-500 text-white border-transparent',
		}
		return map[severity] ?? 'bg-slate-500 hover:bg-slate-500 text-white border-transparent'
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Events</h1>
			<p class="text-muted-foreground">
				Inspect ingested security events from agents
			</p>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<div class="relative flex-1 min-w-48 max-w-xs">
			<SearchIcon
				class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none"
			/>
			<Input
				class="pl-9"
				placeholder="Search events…"
				value={params.search}
				oninput={(e) =>
					updateTableParams(
						{ search: (e.target as HTMLInputElement).value },
						$page.url,
					)}
			/>
		</div>

		<Select.Root
			type="single"
			value={params.severity}
			onValueChange={(v) => updateExtraParams({ severity: String(v ?? '') })}
		>
			<Select.Trigger class="w-42.5"
				>{params.severity || 'All severities'}</Select.Trigger
			>
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
			class="w-42.5"
			placeholder="Category"
			value={params.category}
			onchange={(e) =>
				updateExtraParams({ category: (e.target as HTMLInputElement).value })}
		/>

		<Input
			class="w-55"
			placeholder="Agent ID"
			value={params.agent_id}
			onchange={(e) =>
				updateExtraParams({ agent_id: (e.target as HTMLInputElement).value })}
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
					<Button variant="ghost" size="sm" onclick={clearDateRange}
						>Clear</Button
					>
					<Button variant="default" size="sm" onclick={applyDateRange}
						>Apply</Button
					>
				</div>
			</DropdownMenu.Content>
		</DropdownMenu.Root>

		{#if metadata}
			<span class="ml-auto text-sm text-muted-foreground">
				{events.length} event{events.length !== 1 ? 's' : ''} on this page
			</span>
		{/if}
		<span
			class="flex items-center gap-1.5 text-xs text-muted-foreground"
			title="Realtime stream status"
		>
			<span
				class="inline-block h-2 w-2 rounded-full {sseStatus === 'connected'
					? 'bg-green-500'
					: sseStatus === 'connecting'
						? 'bg-yellow-500 animate-pulse'
						: 'bg-red-500'}"
			></span>
			{sseStatus === 'connected'
				? 'Live'
				: sseStatus === 'connecting'
					? 'Connecting'
					: 'Offline'}
		</span>
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		{#if eventsQuery.isPending}
			<Loading label="Loading events…" />
		{:else if eventsQuery.isError}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span class="text-destructive font-medium">Failed to load events</span>
				<span>{eventsQuery.error?.message}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => eventsQuery.refetch()}>Try again</Button
				>
			</div>
		{:else if events.length === 0}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span>No events found</span>
			</div>
		{:else}
			<EventsDataTable
				data={events}
				params={tableParams}
				onSortChange={() => {}}
				onView={viewEvent}
			/>
		{/if}
	</div>

	<div class="flex items-center justify-between">
		<LimitSelect
			value={params.limit}
			onValueChange={(v) => handleLimitChange(String(v))}
		/>
		<div class="flex items-center gap-2">
			<span class="text-sm text-muted-foreground">Page {currentPage}</span>
			<Button
				variant="outline"
				size="icon"
				class="h-8 w-8"
				disabled={cursorStack.length <= 1 || eventsQuery.isFetching}
				onclick={goPrev}
				aria-label="Previous page"
			>
				<ChevronLeftIcon class="h-4 w-4" />
			</Button>
			<Button
				variant="outline"
				size="icon"
				class="h-8 w-8"
				disabled={!hasMore || eventsQuery.isFetching}
				onclick={goNext}
				aria-label="Next page"
			>
				<ChevronRightIcon class="h-4 w-4" />
			</Button>
		</div>
	</div>

	<Dialog.Root bind:open={detailDialogOpen}>
		<Dialog.Content size="xl" class="max-h-[85vh] flex flex-col overflow-hidden">
			<Dialog.Header class="shrink-0">
				<Dialog.Title>Event Details</Dialog.Title>
				<Dialog.Description
					>Inspect complete payload for the selected event.</Dialog.Description
				>
			</Dialog.Header>

			{#if selectedEventID}
				<div class="min-h-0 flex-1 overflow-y-auto px-0.5">
				{#if detailQuery.isPending}
					<Loading label="Loading event details…" />
				{:else if detailQuery.isError}
					<div class="p-6 text-sm text-destructive">
						Failed to load event details: {detailQuery.error?.message}
					</div>
				{:else if !detailQuery.data?.data}
					<div class="p-6 text-sm text-muted-foreground">
						Event detail not found.
					</div>
				{:else}
					{@const detail = detailQuery.data.data}
					{@const relatedTimeline = toEventTimeline(
						events.filter((event) => {
							if (detail.source_ip && event.source_ip === detail.source_ip)
								return true
							if (detail.agent_id && event.agent_id === detail.agent_id)
								return true
							if (detail.hostname && event.hostname === detail.hostname)
								return true
							return false
						}),
					).slice(0, 8)}
					<Tabs.Root value="identity" class="w-full">
							<Tabs.List class="w-full">
								<Tabs.Trigger value="identity" class="flex-1 gap-2"
									><FingerprintIcon class="h-4 w-4" />Identity</Tabs.Trigger
								>
								<Tabs.Trigger value="timestamps" class="flex-1 gap-2"
									><ClockIcon class="h-4 w-4" />Timestamps</Tabs.Trigger
								>
								<Tabs.Trigger value="payload" class="flex-1 gap-2"
									><BracesIcon class="h-4 w-4" />Payload</Tabs.Trigger
								>
							</Tabs.List>

							<Tabs.Content value="identity" class="mt-4">
								<div class="rounded-lg border divide-y text-sm overflow-hidden">
									<div class="flex items-start gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Event ID</span>
										<span class="font-mono text-xs break-all select-all">{detail.id}</span>
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Agent ID</span>
										<a
											class="font-mono text-xs underline underline-offset-2 hover:text-foreground/80 break-all"
											href={buildEventsFilterPivotHref({ agent_id: detail.agent_id })}
										>{detail.agent_id}</a>
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Hostname</span>
										{#if detail.hostname}
											<a
												class="underline underline-offset-2 hover:text-foreground/80"
												href={buildEventsSearchPivotHref(detail.hostname)}
											>{detail.hostname}</a>
										{:else}
											<span class="text-muted-foreground">—</span>
										{/if}
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Source IP</span>
										{#if detail.source_ip}
											<a
												class="font-mono text-xs underline underline-offset-2 hover:text-foreground/80"
												href={buildEventsSearchPivotHref(detail.source_ip)}
											>{detail.source_ip}</a>
										{:else}
											<span class="text-muted-foreground">—</span>
										{/if}
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Severity</span>
									{#if detail.severity}
										<Badge class={severityClass(detail.severity)}>{detail.severity}</Badge>
										{:else}
											<span class="text-muted-foreground">—</span>
										{/if}
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Category</span>
										{#if detail.category}
											<Badge variant="outline" class="hover:bg-accent transition-colors">
												<a
													href={buildEventsFilterPivotHref({ category: detail.category })}
													class="hover:text-foreground/80"
												>{detail.category}</a>
											</Badge>
										{:else}
											<span class="text-muted-foreground">—</span>
										{/if}
									</div>
								</div>
							</Tabs.Content>

							<Tabs.Content value="timestamps" class="mt-4">
								<div class="rounded-lg border divide-y text-sm overflow-hidden">
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Event Time</span>
										<span class="font-medium">{formatDateTimeExact(detail.event_time)}</span>
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Received At</span>
										<span class="font-medium">{formatDateTimeExact(detail.received_at)}</span>
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Input Type</span>
										<Badge variant="outline" class="font-mono text-xs">{detail.input_type || '—'}</Badge>
									</div>
									<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
										<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Facility</span>
										<Badge variant="outline" class="font-mono text-xs">{detail.facility || '—'}</Badge>
									</div>
									<div class="px-4 py-3">
										<div class="flex gap-3">
											<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium pt-0.5">Message</span>
											<div class="rounded-lg bg-muted/40 border px-3 py-2 text-xs font-mono leading-relaxed wrap-break-word flex-1 min-w-0">
												{detail.message}
											</div>
										</div>
									</div>
									{#if detail.source_ip}
										<div class="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/20 transition-colors">
											<span class="w-30 shrink-0 text-muted-foreground text-xs font-medium">Related</span>
											<a
												class="inline-flex items-center gap-1.5 text-xs font-medium text-primary underline underline-offset-2 hover:text-primary/80 transition-colors"
												href={buildAlertsPivotHref(detail.source_ip)}
											>
												Open alerts by source IP
											</a>
										</div>
									{/if}
								</div>
							</Tabs.Content>

							<Tabs.Content value="payload" class="mt-4">
								<div class="flex flex-col gap-4">
									<div class="rounded-lg border overflow-hidden">
										<div class="flex items-center justify-between border-b bg-muted/30 px-4 py-2">
											<div class="flex items-center gap-2">
												<span class="text-xs font-medium text-muted-foreground">Normalized</span>
												{#if detail.normalized && Object.keys(detail.normalized).length > 0}
													<span class="rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">{Object.keys(detail.normalized).length} keys</span>
												{/if}
											</div>
										</div>
										<pre
											class="text-xs max-h-56 overflow-auto p-4 whitespace-pre-wrap wrap-break-word bg-muted/10 font-mono leading-relaxed">{stringifyNormalized(
												detail.normalized,
											)}</pre>
									</div>
									<div class="rounded-lg border overflow-hidden">
										<div class="flex items-center justify-between border-b bg-muted/30 px-4 py-2">
											<span class="text-xs font-medium text-muted-foreground">Raw</span>
										</div>
										<pre
											class="text-xs max-h-56 overflow-auto p-4 whitespace-pre-wrap wrap-break-word bg-muted/10 font-mono leading-relaxed">{detail.raw ||
											'—'}</pre>
									</div>
								</div>
							</Tabs.Content>
						</Tabs.Root>

						<!-- Related timeline always visible below tabs -->
						<div class="rounded-lg border overflow-hidden mt-4">
							<div class="flex items-center border-b bg-muted/30 px-4 py-2">
								<div class="flex items-center gap-2">
									<ActivityIcon class="h-3.5 w-3.5 text-muted-foreground" />
									<span class="text-xs font-medium text-muted-foreground">
										Related timeline (current result set)
									</span>
								</div>
								{#if relatedTimeline.length > 0}
									<span class="ml-auto text-xs text-muted-foreground">{relatedTimeline.length} event{relatedTimeline.length !== 1 ? 's' : ''}</span>
								{/if}
							</div>
							<div class="p-4">
								{#if relatedTimeline.length === 0}
									<p class="text-xs text-muted-foreground text-center py-4">
										No related events in current page result.
									</p>
								{:else}
									<ul class="relative ml-2 border-l-2 border-muted-foreground/20 space-y-4">
										{#each relatedTimeline as item (item.id + (item.event_time || item.received_at || ''))}
											<li class="relative pl-5 -ml-px">
												<span class="absolute left-0 top-1.5 -translate-x-1/2 h-2.5 w-2.5 rounded-full border-2 border-muted-foreground/40 bg-background"></span>
												<div class="rounded-lg border bg-muted/20 px-3 py-2 hover:bg-muted/30 transition-colors">
													<span class="block text-[10px] text-muted-foreground font-medium mb-0.5">
														{formatDateTime(item.event_time || item.received_at)}
													</span>
													<span class="text-xs wrap-break-word leading-relaxed">{item.message}</span>
												</div>
											</li>
										{/each}
									</ul>
								{/if}
							</div>
						</div>
					{/if}
				</div>
			{/if}
		</Dialog.Content>
	</Dialog.Root>
</div>

<svelte:head>
	<title>Xemarify - Events</title>
</svelte:head>
