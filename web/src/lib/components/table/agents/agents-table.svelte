<script lang="ts">
	import type { Agent } from '$lib/types/api'
	import type { ColumnDef, RowSelectionState } from '@tanstack/table-core'
	import {
		renderComponent,
		renderSnippet,
	} from '$lib/components/ui/data-table/index.js'
	import DataTable from '$lib/components/custom/data-table/data-table.svelte'
	import { cellSnippet } from '$lib/components/custom/data-table/cell-snippet'
	import AgentStatusBadge from '$lib/components/table/agents/agent-status-badge.svelte'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import AgentRowActions from '$lib/components/table/agents/agent-row-actions.svelte'
	import type { TableParams } from '$lib/constant'

	let {
		data,
		params,
		rowSelection = $bindable({}),
		onSortChange,
		onDelete,
		onEdit,
	}: {
		data: Agent[]
		params: TableParams
		rowSelection: RowSelectionState
		onSortChange: (sort: string, order: 'asc' | 'desc') => void
		onDelete: (id: string) => void
		onEdit: (id: string, data: {
			name: string
			hostname?: string
			ip_address?: string
			version?: string
			status: Agent['status']
		}) => void
	} = $props()

	const columns: ColumnDef<Agent>[] = [
		{
			id: 'select',
			enableSorting: false,
			header: () => null,
			cell: () => null,
		},
		{
			id: 'name',
			accessorKey: 'name',
			header: 'Agent Name',
			enableSorting: true,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, {
					value: row.original.name,
					class: 'font-medium',
				}),
		},
		{
			id: 'hostname',
			accessorKey: 'hostname',
			header: 'Hostname',
			enableSorting: true,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, { value: row.original.hostname ?? '—' }),
		},
		{
			id: 'ip_address',
			accessorKey: 'ip_address',
			header: 'IP Address',
			enableSorting: false,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, {
					value: row.original.ip_address ?? '—',
					class: 'font-mono',
				}),
		},
		{
			id: 'status',
			accessorKey: 'status',
			header: 'Status',
			enableSorting: false,
			cell: ({ row }) =>
				renderComponent(AgentStatusBadge, { status: row.original.status }),
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
				renderComponent(AgentRowActions, { agent: row.original, onDelete, onEdit }),
		},
	]
</script>

<DataTable
	{data}
	{columns}
	{params}
	bind:rowSelection
	enableRowSelection
	onSortChange={onSortChange}
/>
