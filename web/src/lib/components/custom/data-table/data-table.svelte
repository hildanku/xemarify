<script lang="ts" generics="T">
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
	} from '$lib/components/ui/data-table/index.js'
	import * as Table from '$lib/components/ui/table/index.js'
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up'
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down'
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down'
	import type { TableParams } from '$lib/constant'

	let {
		data,
		columns,
		params,
		rowSelection = $bindable({}),
		enableRowSelection = false,
		getRowId = (row: T) => (row as any).id,
		onSortChange,
		actionsColumnWidth = 'w-12',
	}: {
		data: T[]
		columns: ColumnDef<T>[]
		params: TableParams
		rowSelection?: RowSelectionState
		enableRowSelection?: boolean
		getRowId?: (row: T) => string
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
		actionsColumnWidth?: string
	} = $props()

	// Sorting state derived from params
	const sorting = $derived<SortingState>([
		{ id: params.sort, desc: params.order === 'desc' },
	])

	// Table instance
	const table = createSvelteTable<T>({
		get data() {
			return data
		},
		columns,
		state: {
			get sorting() {
				return sorting
			},
			get rowSelection() {
				return rowSelection
			},
			get pagination() {
				return { pageIndex: params.page - 1, pageSize: params.limit }
			},
		},
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		manualPagination: true,
		manualSorting: true,
		enableRowSelection,
		getRowId,
		onRowSelectionChange(updater) {
			rowSelection = typeof updater === 'function' ? updater(rowSelection) : updater
		},
		onSortingChange(updater) {
			const next = typeof updater === 'function' ? updater(sorting) : updater
			const first = next[0]
			if (first) onSortChange(first.id, first.desc ? 'desc' : 'asc')
		},
	})

	// Row selection helpers
	const allSelected = $derived(
		enableRowSelection && data.length > 0 && data.every((item) => rowSelection[getRowId(item)]),
	)
	const someSelected = $derived(
		enableRowSelection && data.some((item) => rowSelection[getRowId(item)]) && !allSelected,
	)

	function toggleSelectAll() {
		if (allSelected) {
			rowSelection = {}
		} else {
			const next: RowSelectionState = {}
			data.forEach((item) => (next[getRowId(item)] = true))
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
					<Table.Head
						class={header.column.id === 'actions' ? `${actionsColumnWidth} text-right` : ''}
					>
						{#if header.column.id === 'select' && enableRowSelection}
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
								<FlexRender
									content={header.column.columnDef.header}
									context={header.getContext()}
								/>
								{#if getSortIcon(header.column.id) === 'asc'}
									<ChevronUpIcon class="h-4 w-4" />
								{:else if getSortIcon(header.column.id) === 'desc'}
									<ChevronDownIcon class="h-4 w-4" />
								{:else}
									<ChevronsUpDownIcon class="h-4 w-4 text-muted-foreground" />
								{/if}
							</button>
						{:else if header.column.id !== 'actions'}
							<FlexRender
								content={header.column.columnDef.header}
								context={header.getContext()}
							/>
						{/if}
					</Table.Head>
				{/each}
			</Table.Row>
		{/each}
	</Table.Header>

	<Table.Body>
		{#each table.getRowModel().rows as row (row.id)}
			<Table.Row
				data-state={enableRowSelection && rowSelection[row.id] ? 'selected' : undefined}
				class={enableRowSelection && rowSelection[row.id] ? 'bg-muted/50' : ''}
			>
				{#each row.getVisibleCells() as cell (cell.id)}
					<Table.Cell class={cell.column.id === 'actions' ? 'text-right' : ''}>
						{#if cell.column.id === 'select' && enableRowSelection}
							<input
								type="checkbox"
								class="h-4 w-4 rounded border-border cursor-pointer accent-primary"
								checked={!!rowSelection[row.id]}
								onchange={() => toggleRow(row.id)}
								aria-label="Select row"
							/>
						{:else}
							<FlexRender
								content={cell.column.columnDef.cell}
								context={cell.getContext()}
							/>
						{/if}
					</Table.Cell>
				{/each}
			</Table.Row>
		{/each}
	</Table.Body>
</Table.Root>
