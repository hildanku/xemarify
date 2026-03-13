<script lang="ts">
	import type { RuleSeverity } from '$lib/types/api'
	import { cn } from '$lib/utils'

	let { level }: { level: RuleSeverity } = $props()

	const styles: Record<RuleSeverity, string> = {
		INFO: 'border-sky-300 bg-sky-50 text-sky-700 dark:border-sky-700 dark:bg-sky-950 dark:text-sky-400',
		LOW: 'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-700 dark:bg-emerald-950 dark:text-emerald-400',
		MEDIUM: 'border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-400',
		HIGH: 'border-orange-300 bg-orange-50 text-orange-700 dark:border-orange-700 dark:bg-orange-950 dark:text-orange-400',
		CRITICAL: 'border-rose-300 bg-rose-50 text-rose-700 dark:border-rose-700 dark:bg-rose-950 dark:text-rose-400',
	}
</script>

<span
	class={cn(
		'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium',
		styles[level],
	)}
>
	{level}
</span>
