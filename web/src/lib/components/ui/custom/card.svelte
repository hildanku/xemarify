<script lang="ts">
	import { Card, CardContent } from "$lib/components/ui/card";
	import type { ComponentType } from "svelte";

	let { title, value, icon, colorClass }: Props = $props();

	type Props = {
		title: string;
		value: number;
		icon: ComponentType;
		colorClass: string;
	};
</script>

<Card class="overflow-hidden">
	<CardContent class="p-6">
		<div class="flex items-center justify-between">
			<div class="space-y-1">
				<p class="text-sm font-medium text-muted-foreground">{title}</p>
				<p class="text-3xl font-bold">{value.toLocaleString()}</p>
			</div>
			<div class={["size-12 rounded-lg flex items-center justify-center", colorClass]}>
				<svelte:component this={icon} class="size-6" />
			</div>
		</div>
	</CardContent>
</Card>
