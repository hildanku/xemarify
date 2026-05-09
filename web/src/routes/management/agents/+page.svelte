<script lang="ts">
	import type {
		Agent,
		CreateAgentRequest,
		UpdateAgentRequest,
	} from '$lib/types/api'
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
	import AgentsDataTable from '$lib/components/table/agents/agents-table.svelte'
	import Loading from '$lib/components/ui/custom/loading.svelte'
	import Pagination from '$lib/components/ui/custom/pagination.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import SearchIcon from '@lucide/svelte/icons/search'
	import Trash2Icon from '@lucide/svelte/icons/trash-2'
	import AgentCreateDialog from '$lib/components/table/agents/agent-create-dialog.svelte'
	import { realtimeQueryOptions } from '$lib/utils/realtime-query'

	const queryClient = useQueryClient()
	const params = $derived(parseTableParams($page.url))

	let rowSelection = $state<RowSelectionState>({})
	const selectedIds = $derived(
		Object.keys(rowSelection).filter((k) => rowSelection[k]),
	)

	const agentsQuery = createQuery<ApiResponseWithMetadata<Agent[]>>(() => ({
		queryKey: [
			'agents',
			params.page,
			params.limit,
			params.sort,
			params.order,
			params.search,
		],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Agent[]>>(
				`${V1_BASE_URL}/agents?${buildQueryString(params)}`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const agents = $derived(agentsQuery.data?.data.items ?? [])
	const metadata = $derived(agentsQuery.data?.data.metadata)
	const totalPages = $derived(metadata?.total_pages ?? 1)

	const createAgentMutation = createMutation(() => ({
		mutationFn: (data: CreateAgentRequest) =>
			clientFetch(`${V1_BASE_URL}/agents`, {
				method: 'POST',
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] })
			toast.success('Agent created successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to create agent: ${error.message}`)
		},
	}))

	function handleCreate(data: CreateAgentRequest) {
		createAgentMutation.mutate(data)
	}

	const updateAgentMutation = createMutation(() => ({
		mutationFn: ({ id, data }: { id: string; data: UpdateAgentRequest }) =>
			clientFetch(`${V1_BASE_URL}/agents/${id}`, {
				method: 'PUT',
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] })
			toast.success('Agent updated successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to update agent: ${error.message}`)
		},
	}))

	function handleEdit(id: string, data: UpdateAgentRequest) {
		updateAgentMutation.mutate({ id, data })
	}

	const deleteMutation = createMutation(() => ({
		mutationFn: (id: string) =>
			clientFetch(`${V1_BASE_URL}/agents/${id}`, { method: 'DELETE' }),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] })
			toast.success('Agent deleted successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to delete agent: ${error.message}`)
		},
	}))

	function handleDeleteSingle(id: string) {
		if (!confirm('Delete this agent?')) return
		deleteMutation.mutate(id)
		if (rowSelection[id]) {
			const copy = { ...rowSelection }
			delete copy[id]
			rowSelection = copy
		}
	}

	// Bulk delete - fires parallel DELETE requests for all selected rows
	const bulkDeleteMutation = createMutation(() => ({
		mutationFn: async (ids: string[]) => {
			await Promise.all(
				ids.map((id) =>
					clientFetch(`${V1_BASE_URL}/agents/${id}`, { method: 'DELETE' }),
				),
			)
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agents'] })
			rowSelection = {}
			toast.success('Selected agents deleted successfully')
		},
		onError: (error: Error) => {
			toast.error(`Bulk delete failed: ${error.message}`)
		},
	}))

	function handleBulkDelete() {
		if (selectedIds.length === 0) return
		if (!confirm(`Delete ${selectedIds.length} selected agent(s)?`)) return
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
			<h1 class="text-3xl font-bold tracking-tight">Agents</h1>
			<p class="text-muted-foreground">
				Monitor and manage connected security agents
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="outline" href="/management/agent-onboarding">
				Agent Onboarding
			</Button>
			<Button variant="outline" href="/management/enrollment-tokens">
				Enrollment Tokens
			</Button>
			<AgentCreateDialog
				onCreate={handleCreate}
				isPending={createAgentMutation.isPending}
			/>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<div class="relative flex-1 min-w-48 max-w-xs">
			<SearchIcon
				class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none"
			/>
			<Input
				class="pl-9"
				placeholder="Search agents..."
				value={params.search}
				oninput={(e) =>
					updateTableParams(
						{ search: (e.target as HTMLInputElement).value },
						$page.url,
					)}
			/>
		</div>
		<!-- Bulk delete -->
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
				{metadata.total} agent{metadata.total !== 1 ? 's' : ''} total
			</span>
		{/if}
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		{#if agentsQuery.isPending}
			<Loading label="Loading agents…" />
		{:else if agentsQuery.isError}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span class="text-destructive font-medium">Failed to load agents</span>
				<span>{agentsQuery.error?.message}</span>
				<Button
					variant="outline"
					size="sm"
					onclick={() => agentsQuery.refetch()}
				>
					Try again
				</Button>
			</div>
		{:else if agents.length === 0}
			<div
				class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
			>
				<span>No agents found</span>
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
			<AgentsDataTable
				data={agents}
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
		<!-- {#if (metadata?.total_pages ?? 0) > 1}
			<Pagination
				page={params.page}
				{totalPages}
				onPageChange={gotoPage}
			/>
		{/if} -->
		<Pagination page={params.page} {totalPages} onPageChange={gotoPage} />
	</div>
</div>
