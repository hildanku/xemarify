<script lang="ts">
	import type { UserRole } from '$lib/types/api'
	import { cn } from '$lib/utils'

	let { role }: { role: UserRole } = $props()

	const styles: Record<UserRole, string> = {
		MANAGER:
			'border-purple-300 bg-purple-50 text-purple-700 dark:border-purple-700 dark:bg-purple-950 dark:text-purple-400',
		ANALYST:
			'border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-400',
		VIEWER:
			'border-slate-300 bg-slate-50 text-slate-600 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-400',
	}
</script>

<span
	class={cn(
		'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium',
		styles[role],
	)}
>
	{role}
</span>
