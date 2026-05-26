<script lang="ts">
	import type { User, CreateUserRequest, UpdateUserRequest } from '$lib/types/api'
	import type { ApiResponseWithMetadata } from '$lib/client'
	import type { RowSelectionState } from '@tanstack/svelte-table'
	import {
		createQuery,
		createMutation,
		useQueryClient,
	} from '@tanstack/svelte-query'
	import { toast } from 'svelte-sonner'
	import { page } from '$app/stores'
	import { clientFetch } from '$lib/client'
	import { V1_BASE_URL } from '$lib/constant'
	import {
		parseTableParams,
		updateTableParams,
		buildQueryString,
	} from '$lib/utils/table-params'
	import { createTableHandlers } from '$lib/utils/table-helpers'
	import UsersDataTable from '$lib/components/table/users/users-table.svelte'
	import UserCreateDialog from '$lib/components/table/users/user-create-dialog.svelte'
	import QueryStateWrapper from '$lib/components/custom/query-state-wrapper.svelte'
	import SearchInput from '$lib/components/custom/search-input.svelte'
	import TableFooter from '$lib/components/custom/table-footer.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import Trash2Icon from '@lucide/svelte/icons/trash-2'

	const queryClient = useQueryClient()
	const params = $derived(parseTableParams($page.url))
	const { handleSortChange, gotoPage, handleLimitChange } = createTableHandlers()

	let rowSelection = $state<RowSelectionState>({})
	const selectedIds = $derived(
		Object.keys(rowSelection).filter((k) => rowSelection[k]),
	)

	const usersQuery = createQuery<ApiResponseWithMetadata<User[]>>(() => ({
		queryKey: [
			'users',
			params.page,
			params.limit,
			params.sort,
			params.order,
			params.search,
		],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<User[]>>(
				`${V1_BASE_URL}/users?${buildQueryString(params)}`,
				{ method: 'GET' },
			),
	}))

	const users = $derived(usersQuery.data?.data.items ?? [])
	const metadata = $derived(usersQuery.data?.data.metadata)
	const totalPages = $derived(metadata?.total_pages ?? 1)

	const userMutation = createMutation(() => ({
		mutationFn: (data: CreateUserRequest) =>
			clientFetch(`${V1_BASE_URL}/users`, {
				method: 'POST',
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['users'] })
			toast.success('User created successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to create user: ${error.message}`)
		},
	}))

	function handleCreate(data: {
		username: string
		email: string
		role: string
		password: string
		avatar?: string
	}) {
		userMutation.mutate(data as CreateUserRequest)
	}

	const updateMutation = createMutation(() => ({
		mutationFn: ({ id, data }: { id: string; data: UpdateUserRequest }) =>
			clientFetch(`${V1_BASE_URL}/users/${id}`, {
				method: 'PUT',
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['users'] })
			toast.success('User updated successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to update user: ${error.message}`)
		},
	}))

	function handleEdit(
		id: string,
		data: { username: string; email: string; role: string; avatar?: string },
	) {
		updateMutation.mutate({ id, data: data as UpdateUserRequest })
	}

	const deleteMutation = createMutation(() => ({
		mutationFn: (id: string) =>
			clientFetch(`${V1_BASE_URL}/users/${id}`, { method: 'DELETE' }),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['users'] })
			toast.success('User deleted successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to delete user: ${error.message}`)
		},
	}))

	function handleDeleteSingle(id: string) {
		if (!confirm('Delete this user?')) return
		deleteMutation.mutate(id)
		if (rowSelection[id]) {
			const copy = { ...rowSelection }
			delete copy[id]
			rowSelection = copy
		}
	}

	const bulkDeleteMutation = createMutation(() => ({
		mutationFn: async (ids: string[]) => {
			await Promise.all(
				ids.map((id) =>
					clientFetch(`${V1_BASE_URL}/users/${id}`, { method: 'DELETE' }),
				),
			)
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['users'] })
			rowSelection = {}
			toast.success('Selected users deleted successfully')
		},
		onError: (error: Error) => {
			toast.error(`Bulk delete failed: ${error.message}`)
		},
	}))

	function handleBulkDelete() {
		if (selectedIds.length === 0) return
		if (!confirm(`Delete ${selectedIds.length} selected user(s)?`)) return
		bulkDeleteMutation.mutate(selectedIds)
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<!-- Page header -->
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Users</h1>
			<p class="text-muted-foreground">Manage system users and their roles</p>
		</div>
		<UserCreateDialog
			onCreate={handleCreate}
			isPending={userMutation.isPending}
		/>
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<SearchInput
			placeholder="Search users…"
			value={params.search}
			onInput={(v) => updateTableParams({ search: v }, $page.url)}
		/>

		{#if selectedIds.length > 0}
			<Button
				variant="destructive"
				size="sm"
				onclick={handleBulkDelete}
				disabled={bulkDeleteMutation.isPending}
			>
				<Trash2Icon class="h-4 w-4 mr-2" />
				Delete {selectedIds.length} selected
			</Button>
		{/if}

		{#if metadata}
			<span class="ml-auto text-sm text-muted-foreground">
				{metadata.total} user{metadata.total !== 1 ? 's' : ''} total
			</span>
		{/if}
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		<QueryStateWrapper
			isPending={usersQuery.isPending}
			isError={usersQuery.isError}
			error={usersQuery.error}
			isEmpty={users.length === 0}
			loadingLabel="Loading users…"
			emptyMessage="No users found"
			showClearSearch={!!params.search}
			onRetry={() => usersQuery.refetch()}
			onClearSearch={() => updateTableParams({ search: '' }, $page.url)}
		>
			<UsersDataTable
				data={users}
				{params}
				bind:rowSelection
				onSortChange={handleSortChange}
				onDelete={handleDeleteSingle}
				onEdit={handleEdit}
			/>
		</QueryStateWrapper>
	</div>

	<TableFooter
		page={params.page}
		{totalPages}
		limit={params.limit}
		onPageChange={gotoPage}
		onLimitChange={handleLimitChange}
	/>
</div>
