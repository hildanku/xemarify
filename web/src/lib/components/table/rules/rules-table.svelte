<script lang="ts">
	import type { Rule, UpdateRuleRequest } from '$lib/types/api'
	import type { ColumnDef, RowSelectionState } from '@tanstack/table-core'
	import {
		renderComponent,
		renderSnippet,
	} from '$lib/components/ui/data-table/index.js'
	import DataTable from '$lib/components/custom/data-table/data-table.svelte'
	import { cellSnippet } from '$lib/components/custom/data-table/cell-snippet'
	import RuleLevelBadge from './rule-level-badge.svelte'
	import RuleRowActions from './rule-row-actions.svelte'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import type { TableParams } from '$lib/constant'

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
			id: 'type',
			header: 'Type',
			enableSorting: false,
			cell: ({ row }) =>
				renderSnippet(cellSnippet, {
					value: (row.original.condition.type ?? 'threshold').toUpperCase(),
					class: 'font-mono',
				}),
		},
		{
			id: 'event_type',
			accessorKey: 'condition.event_type',
			header: 'Event Type',
			enableSorting: false,
			cell: ({ row }) => {
				const condition = row.original.condition
				const ruleType = condition.type ?? 'threshold'
				if (ruleType === 'sequence') {
					return renderSnippet(cellSnippet, {
						value: (condition.sequence_steps ?? []).join(' → ') || '—',
						class: 'font-mono',
					})
				}
				if (ruleType === 'correlation') {
					return renderSnippet(cellSnippet, {
						value: (condition.correlation_event_types ?? []).join(', ') || '—',
						class: 'font-mono',
					})
				}
				return renderSnippet(cellSnippet, { value: condition.event_type ?? '—', class: 'font-mono' })
			},
		},
		{
			id: 'logic',
			header: 'Logic',
			enableSorting: false,
			cell: ({ row }) => {
				const condition = row.original.condition
				const ruleType = condition.type ?? 'threshold'
				if (ruleType === 'anomaly') {
					return renderSnippet(cellSnippet, {
						value: `baseline ${condition.baseline_window_sec ?? 0}s × ${condition.spike_factor ?? 0} (min ${condition.anomaly_min_count ?? 0})`,
					})
				}
				if (ruleType === 'correlation') {
					return renderSnippet(cellSnippet, {
						value: `${condition.threshold ?? 0}/${condition.window_sec ?? 0}s, min distinct ${condition.min_distinct_event_types ?? 0}`,
					})
				}
				return renderSnippet(cellSnippet, { value: `${condition.threshold ?? '—'}/${condition.window_sec ?? 0}s` })
			},
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
</script>

<DataTable
	{data}
	{columns}
	{params}
	bind:rowSelection
	enableRowSelection
	onSortChange={onSortChange}
/>
