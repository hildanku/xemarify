<script lang="ts">
	import type { Alert, AlertStatus } from '$lib/types/api'
	import type { ColumnDef } from '@tanstack/table-core'
	import {
		renderComponent,
		renderSnippet,
	} from '$lib/components/ui/data-table/index.js'
	import DataTable from '$lib/components/custom/data-table/data-table.svelte'
	import { cellSnippet } from '$lib/components/custom/data-table/cell-snippet'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import RuleLevelBadge from '$lib/components/table/rules/rule-level-badge.svelte'
	import AlertStatusBadge from './alert-status-badge.svelte'
	import AlertRowActions from './alert-row-actions.svelte'
	import type { TableParams } from '$lib/constant'

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
</script>

<DataTable
	{data}
	{columns}
	{params}
	onSortChange={onSortChange}
	actionsColumnWidth="w-28"
/>
