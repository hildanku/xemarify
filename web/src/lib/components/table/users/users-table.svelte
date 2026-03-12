<script lang="ts">
	import type { User } from '$lib/types/api'
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
	import UserRoleBadge from '$lib/components/table/users/user-role-badge.svelte'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import UserRowActions from '$lib/components/table/users/user-row-actions.svelte'
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
		data: User[]
		params: TableParams
		rowSelection: RowSelectionState
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
		onDelete: (id: string) => void
		onEdit: (id: string, data: { username: string; email: string; role: string; avatar?: string }) => void
	} = $props()

	const cellSnippet = createRawSnippet<[{ value: string; class?: string }]>(
		(getProps) => ({
			render: () =>
				`<span class="text-sm ${getProps().class ?? ''}">${getProps().value}</span>`,
		}),
	)

	const columns: ColumnDef<User>[] = [
		{
			id: 'select',
			enableSorting: false,
			header: () => null,
			cell: () => null,
		},
		{
			id: 'username',
			accessorKey: 'username',
			header: 'Username',
			enableSorting: true,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, {
					value: row.original.username,
					class: 'font-medium',
				}),
		},
		{
			id: 'email',
			accessorKey: 'email',
			header: 'Email',
			enableSorting: true,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, { value: row.original.email }),
		},
		{
			id: 'role',
			accessorKey: 'role',
			header: 'Role',
			enableSorting: true,
			cell: ({ row }) =>
				renderComponent(UserRoleBadge, { role: row.original.role }),
		},
		{
			id: 'created_at',
			accessorKey: 'created_at',
			header: 'Created At',
			enableSorting: true,
			cell: ({ row }) =>
				renderComponent(CompactDate, { dateString: row.original.created_at }),
		},
		{
			id: 'actions',
			enableSorting: false,
			header: '',
			cell: ({ row }) =>
				renderComponent(UserRowActions, {
					user: row.original,
					onDelete,
					onEdit,
				}),
		},
	]

	const sorting = $derived<SortingState>([
		{ id: params.sort, desc: params.order === 'desc' },
	])

	const table = createSvelteTable<User>({
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
				return {
					pageIndex: params.page - 1,
					pageSize: params.limit,
				}
			},
		},
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		manualPagination: true,
		manualSorting: true,
		enableRowSelection: true,
		getRowId: (row) => row.id,
		onRowSelectionChange(updater) {
			if (typeof updater === 'function') {
				rowSelection = updater(rowSelection)
			} else {
				rowSelection = updater
			}
		},
		onSortingChange(updater) {
			const next = typeof updater === 'function' ? updater(sorting) : updater
			const first = next[0]
			if (first) {
				onSortChange(first.id, first.desc ? 'desc' : 'asc')
			}
		},
	})

	const allSelected = $derived(
		data.length > 0 && data.every((u) => rowSelection[u.id]),
	)
	const someSelected = $derived(
		data.some((u) => rowSelection[u.id]) && !allSelected,
	)

	function toggleSelectAll() {
		if (allSelected) {
			rowSelection = {}
		} else {
			const next: RowSelectionState = {}
			data.forEach((u) => (next[u.id] = true))
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
						class={header.column.id === 'actions' ? 'w-12 text-right' : ''}
					>
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
								class="flex items-center gap-1 text-sm font-medium hover:text-foreground transition-colors -ml-1 px-1 py-0.5 rounded hover:bg-muted"
								onclick={header.column.getToggleSortingHandler()}
							>
								<FlexRender
									content={header.column.columnDef.header}
									context={header.getContext()}
								/>
								{#if getSortIcon(header.column.id) === 'both'}
									<ChevronsUpDownIcon class="h-3 w-3 opacity-50" />
								{:else if getSortIcon(header.column.id) === 'asc'}
									<ChevronUpIcon class="h-3 w-3" />
								{:else}
									<ChevronDownIcon class="h-3 w-3" />
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
				data-state={rowSelection[row.id] ? 'selected' : undefined}
				class={rowSelection[row.id] ? 'bg-muted/50' : ''}
			>
				{#each row.getVisibleCells() as cell (cell.id)}
					<Table.Cell>
						{#if cell.column.id === 'select'}
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
