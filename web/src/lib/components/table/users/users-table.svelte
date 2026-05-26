<script lang="ts">
	import type { User } from '$lib/types/api'
	import type { ColumnDef, RowSelectionState } from '@tanstack/table-core'
	import {
		renderComponent,
		renderSnippet,
	} from '$lib/components/ui/data-table/index.js'
	import DataTable from '$lib/components/custom/data-table/data-table.svelte'
	import { cellSnippet } from '$lib/components/custom/data-table/cell-snippet'
	import UserRoleBadge from '$lib/components/table/users/user-role-badge.svelte'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import UserRowActions from '$lib/components/table/users/user-row-actions.svelte'
	import type { TableParams } from '$lib/constant'

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
</script>

<DataTable
	{data}
	{columns}
	{params}
	bind:rowSelection
	enableRowSelection
	onSortChange={onSortChange}
/>
