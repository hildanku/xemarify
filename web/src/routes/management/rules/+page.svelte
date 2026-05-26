<script lang="ts">
	import type { Rule, CreateRuleRequest, UpdateRuleRequest } from '$lib/types/api'
	import type { ApiResponseWithMetadata, ApiResponse } from '$lib/client'
	import type { RowSelectionState } from '@tanstack/svelte-table'
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query'
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
	import RulesDataTable from '$lib/components/table/rules/rules-table.svelte'
	import RuleUpsertDialog from '$lib/components/table/rules/rule-upsert-dialog.svelte'
	import QueryStateWrapper from '$lib/components/custom/query-state-wrapper.svelte'
	import SearchInput from '$lib/components/custom/search-input.svelte'
	import TableFooter from '$lib/components/custom/table-footer.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import Trash2Icon from '@lucide/svelte/icons/trash-2'

	const queryClient = useQueryClient()
	const params = $derived(parseTableParams($page.url))
	const { handleSortChange, gotoPage, handleLimitChange } = createTableHandlers()

	let rowSelection = $state<RowSelectionState>({})
	const selectedIds = $derived(Object.keys(rowSelection).filter((k) => rowSelection[k]))

	const rulesQuery = createQuery<ApiResponseWithMetadata<Rule[]>>(() => ({
		queryKey: ['rules', params.page, params.limit, params.sort, params.order, params.search],
		queryFn: () =>
			clientFetch<ApiResponseWithMetadata<Rule[]>>(`${V1_BASE_URL}/rules?${buildQueryString(params)}`, {
				method: 'GET',
			}),
	}))

	const rules = $derived(rulesQuery.data?.data.items ?? [])
	const metadata = $derived(rulesQuery.data?.data.metadata)
	const totalPages = $derived(metadata?.total_pages ?? 1)

	const createRuleMutation = createMutation(() => ({
		mutationFn: (data: CreateRuleRequest) =>
			clientFetch<ApiResponse<Rule>>(`${V1_BASE_URL}/rules`, {
				method: 'POST',
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['rules'] })
			toast.success('Rule created successfully')
		},
		onError: (error: Error) => toast.error(`Failed to create rule: ${error.message}`),
	}))

	function handleCreate(data: CreateRuleRequest) {
		createRuleMutation.mutate(data)
	}

	const updateMutation = createMutation(() => ({
		mutationFn: ({ id, data }: { id: string; data: UpdateRuleRequest }) =>
			clientFetch<ApiResponse<Rule>>(`${V1_BASE_URL}/rules/${id}`, {
				method: 'PUT',
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['rules'] })
			toast.success('Rule updated successfully')
		},
		onError: (error: Error) => toast.error(`Failed to update rule: ${error.message}`),
	}))

	function handleEdit(id: string, data: UpdateRuleRequest) {
		updateMutation.mutate({ id, data })
	}

	const deleteMutation = createMutation(() => ({
		mutationFn: (id: string) => clientFetch(`${V1_BASE_URL}/rules/${id}`, { method: 'DELETE' }),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['rules'] })
			toast.success('Rule deleted successfully')
		},
		onError: (error: Error) => toast.error(`Failed to delete rule: ${error.message}`),
	}))

	function handleDeleteSingle(id: string) {
		if (!confirm('Delete this rule?')) return
		deleteMutation.mutate(id)
		if (rowSelection[id]) {
			const copy = { ...rowSelection }
			delete copy[id]
			rowSelection = copy
		}
	}

	const bulkDeleteMutation = createMutation(() => ({
		mutationFn: async (ids: string[]) => {
			await Promise.all(ids.map((id) => clientFetch(`${V1_BASE_URL}/rules/${id}`, { method: 'DELETE' })))
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['rules'] })
			rowSelection = {}
			toast.success('Selected rules deleted successfully')
		},
		onError: (error: Error) => toast.error(`Bulk delete failed: ${error.message}`),
	}))

	function handleBulkDelete() {
		if (selectedIds.length === 0) return
		if (!confirm(`Delete ${selectedIds.length} selected rule(s)?`)) return
		bulkDeleteMutation.mutate(selectedIds)
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Detection Rules</h1>
			<p class="text-muted-foreground">Manage threshold, sequence, correlation, and anomaly detection rules</p>
		</div>
		<RuleUpsertDialog mode="create" onCreate={handleCreate} isPending={createRuleMutation.isPending} />
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<SearchInput
			placeholder="Search rules…"
			value={params.search}
			onInput={(v) => updateTableParams({ search: v }, $page.url)}
		/>

		{#if selectedIds.length > 0}
			<Button variant="destructive" size="sm" onclick={handleBulkDelete} disabled={bulkDeleteMutation.isPending}>
				<Trash2Icon class="h-4 w-4 mr-2" />
				Delete {selectedIds.length} selected
			</Button>
		{/if}

		{#if metadata}
			<span class="ml-auto text-sm text-muted-foreground">{metadata.total} rule{metadata.total !== 1 ? 's' : ''} total</span>
		{/if}
	</div>

	<div class="rounded-lg border bg-background overflow-hidden">
		<QueryStateWrapper
			isPending={rulesQuery.isPending}
			isError={rulesQuery.isError}
			error={rulesQuery.error}
			isEmpty={rules.length === 0}
			loadingLabel="Loading rules…"
			emptyMessage="No rules found"
			showClearSearch={!!params.search}
			onRetry={() => rulesQuery.refetch()}
			onClearSearch={() => updateTableParams({ search: '' }, $page.url)}
		>
			<RulesDataTable
				data={rules}
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
