<script lang="ts">
	import type {
		Agent,
		CreateAgentRequest,
		UpdateAgentRequest,
	} from '$lib/types/api'
	import type { ApiResponseWithCursorMetadata } from '$lib/client'
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
	} from '$lib/utils/table-params'
	import AgentsDataTable from '$lib/components/table/agents/agents-table.svelte'
	import QueryStateWrapper from '$lib/components/custom/query-state-wrapper.svelte'
	import SearchInput from '$lib/components/custom/search-input.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left'
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right'
	import Trash2Icon from '@lucide/svelte/icons/trash-2'
	import AgentCreateDialog from '$lib/components/table/agents/agent-create-dialog.svelte'
	import { realtimeQueryOptions } from '$lib/utils/realtime-query'

	const queryClient = useQueryClient()
	const tableParams = $derived(parseTableParams($page.url))

	let cursorStack = $state<string[]>([''])
	const currentCursor = $derived(cursorStack[cursorStack.length - 1])
	const currentPage = $derived(cursorStack.length)

	const filterKey = $derived(
		[tableParams.search, tableParams.limit, tableParams.order].join('|'),
	)
	let prevFilterKey = $state('')
	$effect(() => {
		if (filterKey !== prevFilterKey) {
			prevFilterKey = filterKey
			cursorStack = ['']
		}
	})

	const agentsQuery = createQuery<ApiResponseWithCursorMetadata<Agent[]>>(() => ({
		queryKey: [
			'agents',
			tableParams.search,
			tableParams.limit,
			tableParams.order,
			currentCursor,
		],
		queryFn: () =>
			clientFetch<ApiResponseWithCursorMetadata<Agent[]>>(
				`${V1_BASE_URL}/agents?${buildAgentsQueryString(tableParams, currentCursor)}`,
				{ method: 'GET' },
			),
		...realtimeQueryOptions(),
	}))

	const agents = $derived(agentsQuery.data?.data.items ?? [])
	const metadata = $derived(agentsQuery.data?.data.metadata)
	const hasMore = $derived(metadata?.has_more ?? false)

	function buildAgentsQueryString(p: { search: string; limit: number; order: string }, cursor: string): string {
		const qs = new URLSearchParams()
		qs.set('limit', String(p.limit))
		qs.set('order', p.order)
		if (cursor) qs.set('cursor', cursor)
		if (p.search) qs.set('search', p.search)
		return qs.toString()
	}

	function goNext() {
		const nc = metadata?.next_cursor
		if (!nc) return
		cursorStack = [...cursorStack, nc]
	}

	function goPrev() {
		if (cursorStack.length <= 1) return
		cursorStack = cursorStack.slice(0, -1)
	}

	function handleLimitChange(value: string | undefined) {
		if (!value) return
		updateTableParams({ limit: parseInt(value) }, $page.url)
	}

	let rowSelection = $state<RowSelectionState>({})
	const selectedIds = $derived(
		Object.keys(rowSelection).filter((k) => rowSelection[k]),
	)

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
			value={tableParams.search}
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

		<span class="ml-auto text-sm text-muted-foreground">
			{agents.length} agent{agents.length !== 1 ? 's' : ''} on this page
		</span>
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		<QueryStateWrapper
			isPending={agentsQuery.isPending}
			isError={agentsQuery.isError}
			error={agentsQuery.error}
			isEmpty={agents.length === 0}
			loadingLabel="Loading agents…"
			emptyMessage="No agents found"
			showClearSearch={!!tableParams.search}
			onRetry={() => agentsQuery.refetch()}
			onClearSearch={() => updateTableParams({ search: '' }, $page.url)}
		>
			<AgentsDataTable
				data={agents}
				params={tableParams}
				bind:rowSelection
				onSortChange={() => {}}
				onDelete={handleDeleteSingle}
				onEdit={handleEdit}
			/>
		</QueryStateWrapper>
	</div>

	<div class="flex items-center justify-between">
		<LimitSelect
			value={tableParams.limit}
			onValueChange={(v) => handleLimitChange(String(v))}
		/>
		<div class="flex items-center gap-2">
			<span class="text-sm text-muted-foreground">Page {currentPage}</span>
			<Button
				variant="outline"
				size="icon"
				class="h-8 w-8"
				disabled={cursorStack.length <= 1 || agentsQuery.isFetching}
				onclick={goPrev}
				aria-label="Previous page"
			>
				<ChevronLeftIcon class="h-4 w-4" />
			</Button>
			<Button
				variant="outline"
				size="icon"
				class="h-8 w-8"
				disabled={!hasMore || agentsQuery.isFetching}
				onclick={goNext}
				aria-label="Next page"
			>
				<ChevronRightIcon class="h-4 w-4" />
			</Button>
		</div>
	</div>
</div>

<svelte:head>
	<title>Xemarify - Agents</title>
</svelte:head>