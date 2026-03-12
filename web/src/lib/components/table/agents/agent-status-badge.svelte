<script lang="ts">
	import type { AgentStatus } from '$lib/types/api'
	import { cn } from '$lib/utils'

	let { status }: { status: AgentStatus } = $props()
</script>

<span
	class={cn(
		'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium',
		status === 'ONLINE'
			? 'border-green-300 bg-green-50 text-green-700 dark:border-green-700 dark:bg-green-950 dark:text-green-400'
			: 'border-red-300 bg-red-50 text-red-700 dark:border-red-700 dark:bg-red-950 dark:text-red-400',
	)}
>
	<span
		class={cn(
			'inline-block h-1.5 w-1.5 rounded-full',
			status === 'ONLINE' ? 'bg-green-500' : 'bg-red-500',
		)}
	></span>
	{status}
</span>
