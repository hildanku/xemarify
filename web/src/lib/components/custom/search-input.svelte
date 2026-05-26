<script lang="ts">
	import { Input } from '$lib/components/ui/input/index.js'
	import SearchIcon from '@lucide/svelte/icons/search'

	let {
		placeholder = 'Search...',
		value = '',
		onInput,
	}: {
		placeholder?: string
		value?: string
		onInput: (value: string) => void
	} = $props()
</script>

<div class="relative flex-1 min-w-48 max-w-xs">
	<SearchIcon
		class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none"
	/>
	<Input
		class="pl-9"
		{placeholder}
		{value}
		oninput={(e) => onInput((e.target as HTMLInputElement).value)}
	/>
</div>
