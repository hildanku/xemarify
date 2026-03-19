<script lang="ts">
	import type { EventItem } from '$lib/types/api'
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

	let {
		data,
		params,
		onSortChange,
	}: {
		data: EventItem[]
		params: TableParams
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
	} = $props()

	const cellSnippet = createRawSnippet<[{ value: string; class?: string }]>(
		(getProps) => ({
			render: () => `<span class="text-sm ${getProps().class ?? ''}">${getProps().value}</span>`,
		}),
	)

	const columns: ColumnDef<EventItem>[] = [
		{
			id: 'received_at',
			accessorKey: 'received_at',
			header: 'Received',
			enableSorting: true,
			cell: ({ row }) => renderComponent(CompactDate, { dateString: row.original.received_at }),
		},
		{
			id: 'event_time',
			accessorKey: 'event_time',
			header: 'Event Time',
			enableSorting: true,
			cell: ({ row }) => renderComponent(CompactDate, { dateString: row.original.event_time }),
		},
		{
			id: 'hostname',
			accessorKey: 'hostname',
			header: 'Hostname',
			enableSorting: true,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.hostname || '-' }),
		},
		{
			id: 'agent_id',
			accessorKey: 'agent_id',
			header: 'Agent ID',
			enableSorting: false,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.agent_id, class: 'font-mono text-xs' }),
		},
		{
			id: 'severity',
			accessorKey: 'severity',
			header: 'Severity',
			enableSorting: true,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.severity || '-' }),
		},
		{
			id: 'category',
			accessorKey: 'category',
			header: 'Category',
			enableSorting: true,
			cell: ({ row }) => renderSnippet(cellSnippet, { value: row.original.category || '-' }),
		},
		{
			id: 'message',
			accessorKey: 'message',
			header: 'Message',
			enableSorting: false,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, {
					value: row.original.message,
					class: 'max-w-[560px] block truncate',
				}),
		},
	]

	const sorting = $derived<SortingState>([{ id: params.sort, desc: params.order === 'desc' }])

	const table = createSvelteTable<EventItem>({
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
					<Table.Head>
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
						<Table.Cell>
							<FlexRender content={cell.column.columnDef.cell} context={cell.getContext()} />
						</Table.Cell>
					{/each}
				</Table.Row>
			{/each}
		{/if}
	</Table.Body>
</Table.Root>
