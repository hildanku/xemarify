<script lang="ts">
	import { cn } from '$lib/utils'
	import { Button } from '$lib/components/ui/button/index.js'
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left'
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right'

	let {
		page = 1,
		totalPages = 1,
		class: className = '',
		onPageChange,
	}: {
		page: number
		totalPages: number
		class?: string
		onPageChange: (page: number) => void
	} = $props()

	const canPrev = $derived(page > 1)
	const canNext = $derived(page < totalPages)
</script>

<div class={cn('flex items-center gap-1', className)}>
	<Button
		variant="outline"
		size="icon"
		class="h-8 w-8"
		disabled={!canPrev}
		onclick={() => onPageChange(page - 1)}
		aria-label="Previous page"
	>
		<ChevronLeftIcon class="h-4 w-4" />
	</Button>

	<div
		class="flex h-8 min-w-8 items-center justify-center rounded-md border px-3 text-sm font-medium"
	>
		{page}
	</div>

	<Button
		variant="outline"
		size="icon"
		class="h-8 w-8"
		disabled={!canNext}
		onclick={() => onPageChange(page + 1)}
		aria-label="Next page"
	>
		<ChevronRightIcon class="h-4 w-4" />
	</Button>
</div>
