<script lang="ts">
	import Pagination from '$lib/components/ui/custom/pagination.svelte'
	import LimitSelect from '$lib/components/ui/custom/limit-select.svelte'

	let {
		page,
		totalPages,
		limit,
		onPageChange,
		onLimitChange,
	}: {
		page: number
		totalPages: number
		limit: number
		onPageChange: (page: number) => void
		onLimitChange: (value: string | undefined) => void
	} = $props()
</script>

<div class="flex items-center justify-between">
	<LimitSelect
		value={limit}
		onValueChange={(v) => onLimitChange(String(v))}
	/>
	<Pagination {page} {totalPages} onPageChange={onPageChange} />
</div>
