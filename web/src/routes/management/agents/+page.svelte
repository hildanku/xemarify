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
	import { createTableHandlers } from '$lib/utils/table-helpers'
	import AgentsDataTable from '$lib/components/table/agents/agents-table.svelte'
	import QueryStateWrapper from '$lib/components/custom/query-state-wrapper.svelte'
	import SearchInput from '$lib/components/custom/search-input.svelte'
	import TableFooter from '$lib/components/custom/table-footer.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import Trash2Icon from '@lucide/svelte/icons/trash-2'
	import AgentCreateDialog from '$lib/components/table/agents/agent-create-dialog.svelte'
	import { realtimeQueryOptions } from '$lib/utils/realtime-query'

	const queryClient = useQueryClient()
	const params = $derived(parseTableParams($page.url))
	const { handleSortChange, gotoPage, handleLimitChange } = createTableHandlers()

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
		<SearchInput
			placeholder="Search agents..."
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
				{metadata.total} agent{metadata.total !== 1 ? 's' : ''} total
			</span>
		{/if}
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		<QueryStateWrapper
			isPending={agentsQuery.isPending}
			isError={agentsQuery.isError}
			error={agentsQuery.error}
			isEmpty={agents.length === 0}
			loadingLabel="Loading agents…"
			emptyMessage="No agents found"
			showClearSearch={!!params.search}
			onRetry={() => agentsQuery.refetch()}
			onClearSearch={() => updateTableParams({ search: '' }, $page.url)}
		>
			<AgentsDataTable
				data={agents}
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

<svelte:head>
	<title>Xemarify - Agents</title>
</svelte:head>
