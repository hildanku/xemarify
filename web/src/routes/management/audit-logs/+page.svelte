<script lang="ts">
	import { page } from '$app/stores'
	import { createQuery } from '@tanstack/svelte-query'
	import { clientFetch, type ApiResponseWithMetadata } from '$lib/client'
	import { V1_BASE_URL, type TableParams } from '$lib/constant'
	import {
		parseTableParams,
		updateSearchParams,
		updateTableParams,
	} from '$lib/utils/table-params'
	import type { AuditLog } from '$lib/types/api'
	import Loading from '$lib/components/ui/custom/loading.svelte'
	import Pagination from '$lib/components/ui/custom/pagination.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'
	import AuditLogsTable from '$lib/components/table/audit-logs/audit-logs-table.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import SearchIcon from '@lucide/svelte/icons/search'
	import CalendarIcon from '@lucide/svelte/icons/calendar'
	import { realtimeQueryOptions } from '$lib/utils/realtime-query'

	type AuditLogPageParams = TableParams & {
		action: string
		date_from: string
		date_to: string
	}

	const tableParams = $derived(parseTableParams($page.url))
	const params = $derived(parseAuditLogParams($page.url, tableParams))
	let dateFrom = $state('')
	let dateTo = $state('')

	$effect(() => {
		dateFrom = toDateInputValue(params.date_from)
		dateTo = toDateInputValue(params.date_to)
	})

	const auditLogsQuery = createQuery<ApiResponseWithMetadata<AuditLog[]>>(
		() => ({
			queryKey: ['audit-logs', params],
			queryFn: () =>
				clientFetch<ApiResponseWithMetadata<AuditLog[]>>(
					`${V1_BASE_URL}/audit-logs?${buildAuditLogQueryString(params)}`,
					{ method: 'GET' },
				),
			...realtimeQueryOptions(),
		}),
	)

	const entries = $derived(auditLogsQuery.data?.data.items ?? [])
	const metadata = $derived(auditLogsQuery.data?.data.metadata)
	const totalPages = $derived(metadata?.total_pages ?? 1)

	function parseAuditLogParams(
		url: URL,
		table: TableParams,
	): AuditLogPageParams {
		return {
			...table,
			action: url.searchParams.get('action') ?? '',
			date_from: url.searchParams.get('date_from') ?? '',
			date_to: url.searchParams.get('date_to') ?? '',
		}
	}

	function buildAuditLogQueryString(p: AuditLogPageParams): string {
		const qs = new URLSearchParams()
		qs.set('limit', String(p.limit))
		qs.set('offset', String(Math.max(0, (p.page - 1) * p.limit)))
		qs.set('sort_by', p.sort)
		qs.set('order', p.order)
		if (p.search) qs.set('search', p.search)
		if (p.action) qs.set('action', p.action)
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

	function updateExtraParams(
		next: Partial<Pick<AuditLogPageParams, 'action' | 'date_from' | 'date_to'>>,
	) {
		const resetPage =
			('action' in next && next.action !== params.action) ||
			('date_from' in next && next.date_from !== params.date_from) ||
			('date_to' in next && next.date_to !== params.date_to)

		updateSearchParams(
			{
				action: next.action,
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
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Audit Logs</h1>
			<p class="text-muted-foreground">
				Review authentication and management actions across the system
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
				placeholder="Search action or user..."
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
			value={params.action}
			onValueChange={(v) => updateExtraParams({ action: String(v ?? '') })}
		>
			<Select.Trigger class="w-[180px]"
				>{params.action || 'All actions'}</Select.Trigger
			>
			<Select.Content>
				<Select.Item value="">All actions</Select.Item>
				<Select.Item value="LOGIN">LOGIN</Select.Item>
				<Select.Item value="LOGOUT">LOGOUT</Select.Item>
				<Select.Item value="CREATE_USER">CREATE_USER</Select.Item>
				<Select.Item value="UPDATE_USER">UPDATE_USER</Select.Item>
				<Select.Item value="DELETE_USER">DELETE_USER</Select.Item>
			</Select.Content>
		</Select.Root>

		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Button variant="outline" size="sm" {...props}>
						<CalendarIcon class="mr-2 h-4 w-4" />
						Date range
					</Button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align="end" class="w-[320px] space-y-3 p-3">
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
				{metadata.total} audit log{metadata.total !== 1 ? 's' : ''} total
			</span>
		{/if}
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		{#if auditLogsQuery.isPending}
			<Loading label="Loading audit logs..." />
		{:else if auditLogsQuery.isError}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span class="text-destructive font-medium"
					>Failed to load audit logs</span
				>
				<span>{auditLogsQuery.error?.message}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => auditLogsQuery.refetch()}>Try again</Button
				>
			</div>
		{:else if entries.length === 0}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span>No audit logs found</span>
				{#if params.search || params.action || params.date_from || params.date_to}
					<Button
						variant="ghost"
						size="sm"
						onclick={() =>
							updateSearchParams(
								{ search: '', action: '', date_from: '', date_to: '' },
								$page.url,
								{ resetPage: true },
							)}
					>
						Clear filters
					</Button>
				{/if}
			</div>
		{:else}
			<AuditLogsTable data={entries} {params} {onSortChange} />
		{/if}
	</div>

	<div class="flex items-center justify-between">
		<LimitSelect
			value={params.limit}
			onValueChange={(v) => handleLimitChange(String(v))}
		/>
		<Pagination page={params.page} {totalPages} onPageChange={gotoPage} />
	</div>
</div>
