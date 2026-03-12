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
	import UsersDataTable from '$lib/components/table/users/users-table.svelte'
	import UserCreateDialog from '$lib/components/table/users/user-create-dialog.svelte'
	import Loading from '$lib/components/ui/custom/loading.svelte'
	import Pagination from '$lib/components/ui/custom/pagination.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import SearchIcon from '@lucide/svelte/icons/search'
	import Trash2Icon from '@lucide/svelte/icons/trash-2'

	const queryClient = useQueryClient()
	const params = $derived(parseTableParams($page.url))

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

	function handleSortChange(sort: string, order: 'asc' | 'desc') {
		updateTableParams({ sort, order }, $page.url)
	}

	function gotoPage(p: number) {
		updateTableParams({ page: p }, $page.url)
	}

	function handleLimitChange(value: string | undefined) {
		if (!value) return
		updateTableParams({ limit: parseInt(value), page: 1 }, $page.url)
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
		<div class="relative flex-1 min-w-48 max-w-xs">
			<SearchIcon
				class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none"
			/>
			<Input
				class="pl-9"
				placeholder="Search users…"
				value={params.search}
				oninput={(e) =>
					updateTableParams(
						{ search: (e.target as HTMLInputElement).value },
						$page.url,
					)}
			/>
		</div>

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
		{#if usersQuery.isPending}
			<Loading label="Loading users…" />
		{:else if usersQuery.isError}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span class="text-destructive font-medium">Failed to load users</span>
				<span>{usersQuery.error?.message}</span>
				<Button variant="outline" size="sm" onclick={() => usersQuery.refetch()}>
					Try again
				</Button>
			</div>
		{:else if users.length === 0}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span>No users found</span>
				{#if params.search}
					<Button
						variant="ghost"
						size="sm"
						onclick={() => updateTableParams({ search: '' }, $page.url)}
					>
						Clear search
					</Button>
				{/if}
			</div>
		{:else}
			<UsersDataTable
				data={users}
				{params}
				bind:rowSelection
				onSortChange={handleSortChange}
				onDelete={handleDeleteSingle}
				onEdit={handleEdit}
			/>
		{/if}
	</div>

	<div class="flex items-center justify-between">
		<LimitSelect
			value={params.limit}
			onValueChange={(v) => handleLimitChange(String(v))}
		/>
		<Pagination page={params.page} {totalPages} onPageChange={gotoPage} />
	</div>
</div>
