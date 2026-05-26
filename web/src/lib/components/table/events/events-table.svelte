<script lang="ts">
	import type { EventItem } from '$lib/types/api'
	import type { ColumnDef } from '@tanstack/table-core'
	import {
		renderComponent,
		renderSnippet,
	} from '$lib/components/ui/data-table/index.js'
	import DataTable from '$lib/components/custom/data-table/data-table.svelte'
	import { cellSnippet } from '$lib/components/custom/data-table/cell-snippet'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import EventRowActions from './event-row-actions.svelte'
	import type { TableParams } from '$lib/constant'

	let {
		data,
		params,
		onSortChange,
		onView,
	}: {
		data: EventItem[]
		params: TableParams
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
		onView: (id: string) => void
	} = $props()

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
		{
			id: 'actions',
			enableSorting: false,
			header: '',
			cell: ({ row }) => renderComponent(EventRowActions, { event: row.original, onView }),
		},
	]
</script>

<DataTable
	{data}
	{columns}
	{params}
	onSortChange={onSortChange}
	actionsColumnWidth="w-28"
/>
