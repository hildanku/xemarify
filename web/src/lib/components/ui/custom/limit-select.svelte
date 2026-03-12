<script lang="ts">
	import { cn } from '$lib/utils'
	import * as Select from '$lib/components/ui/select/index.js'
	import { LIMIT_OPTIONS } from '$lib/constant'

	let {
		value = 10,
		options = LIMIT_OPTIONS as unknown as number[],
		class: className = '',
		onValueChange,
	}: {
		value?: number
		options?: number[]
		class?: string
		onValueChange: (value: number) => void
	} = $props()
</script>

<Select.Root
	type="single"
	value={String(value)}
	onValueChange={(v) => {
		if (!v) return
		const parsed = parseInt(v)
		if (!isNaN(parsed)) onValueChange(parsed)
	}}
>
	<Select.Trigger class={cn('w-[110px]', className)}>
		<span>Limit: {value}</span>
	</Select.Trigger>
	<Select.Content>
		{#each options as opt (opt)}
			<Select.Item value={String(opt)}>{opt} rows</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>
