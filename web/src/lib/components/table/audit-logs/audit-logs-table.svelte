<script lang="ts">
	import type { AuditLog } from '$lib/types/api'
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
	import AuditLogActionBadge from './audit-log-action-badge.svelte'
	import AuditLogRowActions from './audit-log-row-actions.svelte'

	let {
		data,
		params,
		onSortChange,
	}: {
		data: AuditLog[]
		params: TableParams
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
	} = $props()

	const cellSnippet = createRawSnippet<[{ value: string; class?: string }]>(
		(getProps) => ({
			render: () =>
				`<span class="text-sm ${getProps().class ?? ''}">${getProps().value}</span>`,
		}),
	)

	const columns: ColumnDef<AuditLog>[] = [
		{
			id: 'created_at',
			accessorKey: 'created_at',
			header: 'Created',
			enableSorting: true,
			cell: ({ row }) =>
				renderComponent(CompactDate, { dateString: row.original.created_at }),
		},
		{
			id: 'action',
			accessorKey: 'action',
			header: 'Action',
			enableSorting: true,
			cell: ({ row }) =>
				renderComponent(AuditLogActionBadge, { action: row.original.action }),
		},
		{
			id: 'user_identifier',
			accessorKey: 'user_identifier',
			header: 'User',
			enableSorting: true,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, {
					value: row.original.user_identifier,
					class: 'font-medium break-all',
				}),
		},
		{
			id: 'object_type',
			accessorKey: 'object_type',
			header: 'Object Type',
			enableSorting: false,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, { value: row.original.object_type ?? '—' }),
		},
		{
			id: 'object_id',
			accessorKey: 'object_id',
			header: 'Object ID',
			enableSorting: false,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, {
					value: row.original.object_id ?? '—',
					class: 'font-mono text-xs',
				}),
		},
		{
			id: 'actions',
			enableSorting: false,
			header: '',
			cell: ({ row }) => renderComponent(AuditLogRowActions, { entry: row.original }),
		},
	]

	const sorting = $derived<SortingState>([
		{ id: params.sort, desc: params.order === 'desc' },
	])

	const table = createSvelteTable<AuditLog>({
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
					<Table.Head class={header.column.id === 'actions' ? 'w-24 text-right' : ''}>
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
