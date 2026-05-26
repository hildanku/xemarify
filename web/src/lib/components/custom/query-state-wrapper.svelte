<script lang="ts">
	import Loading from '$lib/components/ui/custom/loading.svelte'
	import { Button } from '$lib/components/ui/button/index.js'
	import type { Snippet } from 'svelte'

	let {
		isPending,
		isError,
		error,
		isEmpty,
		loadingLabel = 'Loading...',
		emptyMessage = 'No items found',
		showClearSearch = false,
		onRetry,
		onClearSearch,
		children,
	}: {
		isPending: boolean
		isError: boolean
		error?: Error | null
		isEmpty: boolean
		loadingLabel?: string
		emptyMessage?: string
		showClearSearch?: boolean
		onRetry: () => void
		onClearSearch?: () => void
		children: Snippet
	} = $props()
</script>

{#if isPending}
	<Loading label={loadingLabel} />
{:else if isError}
	<div
		class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
	>
		<span class="text-destructive font-medium">Failed to load data</span>
		<span>{error?.message}</span>
		<Button variant="outline" size="sm" onclick={onRetry}>
			Try again
		</Button>
	</div>
{:else if isEmpty}
	<div
		class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
	>
		<span>{emptyMessage}</span>
		{#if showClearSearch && onClearSearch}
			<Button variant="ghost" size="sm" onclick={onClearSearch}>
				Clear search
			</Button>
		{/if}
	</div>
{:else}
	{@render children()}
{/if}
