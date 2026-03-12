<script lang="ts">
	import { cn } from '$lib/utils'

	let {
		class: className = '',
		size = 'md',
		label = 'Loading…',
	}: {
		class?: string
		size?: 'sm' | 'md' | 'lg'
		label?: string
	} = $props()

	const sizeClasses = {
		sm: 'h-4 w-4 border-2',
		md: 'h-8 w-8 border-2',
		lg: 'h-12 w-12 border-4',
	}
</script>

<div
	class={cn('flex flex-col items-center justify-center gap-3 py-8', className)}
	role="status"
	aria-label={label}
>
	<div
		class={cn(
			'animate-spin rounded-full border-muted border-t-primary',
			sizeClasses[size],
		)}
	></div>
	<span class="text-sm text-muted-foreground">{label}</span>
</div>
