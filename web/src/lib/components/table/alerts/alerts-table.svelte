<script lang="ts">
	import type { Alert, AlertStatus } from '$lib/types/api'
	import {
		getCoreRowModel,
		getPaginationRowModel,
		type ColumnDef,
		type SortingState,
	} from '@tanstack/table-core'
	import {
		createSvelteTable,
		FlexRender,
		renderComponent,
		renderSnippet,
	} from '$lib/components/ui/data-table/index.js'
	import * as Table from '$lib/components/ui/table/index.js'
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up'
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down'
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down'
	import type { TableParams } from '$lib/constant'
	import { createRawSnippet } from 'svelte'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import RuleLevelBadge from '$lib/components/table/rules/rule-level-badge.svelte'
	import AlertStatusBadge from './alert-status-badge.svelte'
	import AlertRowActions from './alert-row-actions.svelte'

	let {
		data,
		params,
		onSortChange,
		onView,
		onStatus,
	}: {
		data: Alert[]
		params: TableParams
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
		onView: (id: string) => void
		onStatus: (id: string, status: AlertStatus) => void
	} = $props()

	const cellSnippet = createRawSnippet<[{ value: string; class?: string }]>(
		(getProps) => ({
			render: () => `<span class="text-sm ${getProps().class ?? ''}">${getProps().value}</span>`,
		}),
	)

	const columns: ColumnDef<Alert>[] = [
		{
			id: 'severity',
			accessorKey: 'severity',
			header: 'Severity',
			enableSorting: true,
			cell: ({ row }) => renderComponent(RuleLevelBadge, { level: row.original.severity }),
		},
		{
			id: 'rule_name',
			accessorKey: 'rule_name',
			header: 'Rule Name',
			enableSorting: false,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.rule_name, class: 'font-medium' }),
		},
		{
			id: 'correlation_key',
			accessorKey: 'correlation_key',
			header: 'Correlation Key',
			enableSorting: false,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.correlation_key, class: 'font-mono text-xs' }),
		},
		{
			id: 'triggered_at',
			accessorKey: 'triggered_at',
			header: 'Triggered At',
			enableSorting: true,
			cell: ({ row }) => renderComponent(CompactDate, { dateString: row.original.triggered_at }),
		},
		{
			id: 'status',
			accessorKey: 'status',
			header: 'Status',
			enableSorting: true,
			cell: ({ row }) => renderComponent(AlertStatusBadge, { status: row.original.status }),
		},
		{
			id: 'actions',
			enableSorting: false,
			header: '',
			cell: ({ row }) => renderComponent(AlertRowActions, { alert: row.original, onView, onStatus }),
		},
	]

	const sorting = $derived<SortingState>([{ id: params.sort, desc: params.order === 'desc' }])

	const table = createSvelteTable<Alert>({
		get data() {
			return data
		},
		columns,
		state: {
			get sorting() {
				return sorting
			},
			get pagination() {
				return { pageIndex: params.page - 1, pageSize: params.limit }
			},
		},
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		manualPagination: true,
		manualSorting: true,
		onSortingChange(updater) {
			const next = typeof updater === 'function' ? updater(sorting) : updater
			const first = next[0]
			if (first) onSortChange(first.id, first.desc ? 'desc' : 'asc')
		},
	})

	function getSortIcon(columnId: string) {
		if (params.sort !== columnId) return 'both'
		return params.order === 'asc' ? 'asc' : 'desc'
	}
</script>

<Table.Root>
	<Table.Header>
		{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
			<Table.Row>
				{#each headerGroup.headers as header (header.id)}
					<Table.Head class={header.column.id === 'actions' ? 'w-28 text-right' : ''}>
						{#if header.column.getCanSort()}
							<button
								type="button"
								class="inline-flex items-center gap-1 font-medium hover:text-foreground/80 transition-colors"
								onclick={() => {
									const current = getSortIcon(header.column.id)
									const nextOrder = current === 'asc' ? 'desc' : 'asc'
									onSortChange(header.column.id, nextOrder)
								}}
							>
								<FlexRender content={header.column.columnDef.header} context={header.getContext()} />
								{#if getSortIcon(header.column.id) === 'asc'}
									<ChevronUpIcon class="h-4 w-4" />
								{:else if getSortIcon(header.column.id) === 'desc'}
									<ChevronDownIcon class="h-4 w-4" />
								{:else}
									<ChevronsUpDownIcon class="h-4 w-4 text-muted-foreground" />
								{/if}
							</button>
						{:else}
							<FlexRender content={header.column.columnDef.header} context={header.getContext()} />
						{/if}
					</Table.Head>
				{/each}
			</Table.Row>
		{/each}
	</Table.Header>
	<Table.Body>
		{#if table.getRowModel().rows?.length}
			{#each table.getRowModel().rows as row (row.id)}
				<Table.Row>
					{#each row.getVisibleCells() as cell (cell.id)}
						<Table.Cell class={cell.column.id === 'actions' ? 'text-right' : ''}>
							<FlexRender content={cell.column.columnDef.cell} context={cell.getContext()} />
						</Table.Cell>
					{/each}
				</Table.Row>
			{/each}
		{/if}
	</Table.Body>
</Table.Root>
