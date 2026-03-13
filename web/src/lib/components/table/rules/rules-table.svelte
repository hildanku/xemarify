<script lang="ts">
	import type { Rule, UpdateRuleRequest } from '$lib/types/api'
	import {
		getCoreRowModel,
		getPaginationRowModel,
		type ColumnDef,
		type SortingState,
		type RowSelectionState,
	} from '@tanstack/table-core'
	import {
		createSvelteTable,
		FlexRender,
		renderComponent,
		renderSnippet,
	} from '$lib/components/ui/data-table/index.js'
	import * as Table from '$lib/components/ui/table/index.js'
	import RuleLevelBadge from './rule-level-badge.svelte'
	import RuleRowActions from './rule-row-actions.svelte'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up'
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down'
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down'
	import type { TableParams } from '$lib/constant'
	import { createRawSnippet } from 'svelte'

	let {
		data,
		params,
		rowSelection = $bindable({}),
		onSortChange,
		onDelete,
		onEdit,
	}: {
		data: Rule[]
		params: TableParams
		rowSelection: RowSelectionState
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
		onDelete: (id: string) => void
		onEdit: (id: string, data: UpdateRuleRequest) => void
	} = $props()

	const cellSnippet = createRawSnippet<[{ value: string; class?: string }]>(
		(getProps) => ({
			render: () => `<span class="text-sm ${getProps().class ?? ''}">${getProps().value}</span>`,
		}),
	)

	const columns: ColumnDef<Rule>[] = [
		{ id: 'select', enableSorting: false, header: () => null, cell: () => null },
		{
			id: 'name',
			accessorKey: 'name',
			header: 'Rule Name',
			enableSorting: true,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.name, class: 'font-medium' }),
		},
		{
			id: 'level',
			accessorKey: 'level',
			header: 'Level',
			enableSorting: true,
			cell: ({ row }) => renderComponent(RuleLevelBadge, { level: row.original.level }),
		},
		{
			id: 'event_type',
			accessorKey: 'condition.event_type',
			header: 'Event Type',
			enableSorting: false,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.condition.event_type, class: 'font-mono' }),
		},
		{
			id: 'threshold',
			header: 'Threshold',
			enableSorting: false,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: `${row.original.condition.threshold}/${row.original.condition.window_sec}s` }),
		},
		{
			id: 'enabled',
			accessorKey: 'enabled',
			header: 'Enabled',
			enableSorting: true,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.enabled ? 'Yes' : 'No' }),
		},
		{
			id: 'created_at',
			accessorKey: 'created_at',
			header: 'Created At',
			enableSorting: true,
			cell: ({ row }) => renderComponent(CompactDate, { dateString: row.original.created_at }),
		},
		{
			id: 'actions',
			enableSorting: false,
			header: '',
			cell: ({ row }) => renderComponent(RuleRowActions, { rule: row.original, onDelete, onEdit }),
		},
	]

	const sorting = $derived<SortingState>([{ id: params.sort, desc: params.order === 'desc' }])

	const table = createSvelteTable<Rule>({
		get data() { return data },
		columns,
		state: {
			get sorting() { return sorting },
			get rowSelection() { return rowSelection },
			get pagination() { return { pageIndex: params.page - 1, pageSize: params.limit } },
		},
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		manualPagination: true,
		manualSorting: true,
		enableRowSelection: true,
		getRowId: (row) => row.id,
		onRowSelectionChange(updater) {
			rowSelection = typeof updater === 'function' ? updater(rowSelection) : updater
		},
		onSortingChange(updater) {
			const next = typeof updater === 'function' ? updater(sorting) : updater
			const first = next[0]
			if (first) onSortChange(first.id, first.desc ? 'desc' : 'asc')
		},
	})

	const allSelected = $derived(data.length > 0 && data.every((r) => rowSelection[r.id]))
	const someSelected = $derived(data.some((r) => rowSelection[r.id]) && !allSelected)

	function toggleSelectAll() {
		if (allSelected) {
			rowSelection = {}
		} else {
			const next: RowSelectionState = {}
			data.forEach((r) => (next[r.id] = true))
			rowSelection = next
		}
	}

	function toggleRow(id: string) {
		if (rowSelection[id]) {
			const copy = { ...rowSelection }
			delete copy[id]
			rowSelection = copy
		} else {
			rowSelection = { ...rowSelection, [id]: true }
		}
	}

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
					<Table.Head class={header.column.id === 'actions' ? 'w-12 text-right' : ''}>
						{#if header.column.id === 'select'}
							<input
								type="checkbox"
								class="h-4 w-4 rounded border-border cursor-pointer accent-primary"
								checked={allSelected}
								indeterminate={someSelected}
								onchange={toggleSelectAll}
								aria-label="Select all rows"
							/>
						{:else if header.column.getCanSort()}
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
				<Table.Row data-state={rowSelection[row.id] ? 'selected' : undefined}>
					{#each row.getVisibleCells() as cell (cell.id)}
						<Table.Cell class={cell.column.id === 'actions' ? 'text-right' : ''}>
							{#if cell.column.id === 'select'}
								<input
									type="checkbox"
									class="h-4 w-4 rounded border-border cursor-pointer accent-primary"
									checked={!!rowSelection[row.id]}
									onchange={() => toggleRow(row.id)}
									aria-label={`Select ${row.original.name}`}
								/>
							{:else}
								<FlexRender content={cell.column.columnDef.cell} context={cell.getContext()} />
							{/if}
						</Table.Cell>
					{/each}
				</Table.Row>
			{/each}
		{/if}
	</Table.Body>
</Table.Root>
