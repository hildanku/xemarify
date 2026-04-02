<script lang="ts">
	import type { Rule, UpdateRuleRequest } from '$lib/types/api'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import RuleLevelBadge from './rule-level-badge.svelte'
	import RuleUpsertDialog from '$lib/components/table/rules/rule-upsert-dialog.svelte'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import MoreHorizontalIcon from '@lucide/svelte/icons/more-horizontal'

	let {
		rule,
		onDelete,
		onEdit,
	}: {
		rule: Rule
		onDelete: (id: string) => void
		onEdit: (id: string, data: UpdateRuleRequest) => void
	} = $props()

	let viewOpen = $state(false)
	let editOpen = $state(false)
	const ruleType = $derived(rule.condition.type ?? 'threshold')
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="sm" class="h-8 w-8 p-0" aria-label="Open row actions" {...props}>
				<MoreHorizontalIcon class="h-4 w-4" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-44">
		<DropdownMenu.Item onclick={() => (viewOpen = true)}>View details</DropdownMenu.Item>
		<DropdownMenu.Item onclick={() => (editOpen = true)}>Edit rule</DropdownMenu.Item>
		<DropdownMenu.Separator />
		<DropdownMenu.Item class="text-destructive focus:text-destructive" onclick={() => onDelete(rule.id)}>
			Delete rule
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<RuleUpsertDialog mode="edit" {rule} onEdit={onEdit} bind:open={editOpen} hideTrigger />

<Dialog.Root bind:open={viewOpen}>
	<Dialog.Content class="max-w-2xl">
		<Dialog.Header>
			<Dialog.Title>{rule.name}</Dialog.Title>
			<Dialog.Description>Detection Rule Details</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-2 text-sm">
			<div class="flex items-center gap-2">
				<RuleLevelBadge level={rule.level} />
				<span class="rounded-md border px-2 py-0.5 font-mono text-xs uppercase">{ruleType}</span>
				<span class="text-muted-foreground">{rule.enabled ? 'Enabled' : 'Disabled'}</span>
			</div>
			<div class="grid grid-cols-2 gap-4">
				<div class="col-span-2">
					<p class="font-medium text-muted-foreground">Description</p>
					<p class="mt-0.5">{rule.description || '—'}</p>
				</div>
				{#if ruleType === 'threshold'}
					<div>
						<p class="font-medium text-muted-foreground">Event Type</p>
						<p class="mt-0.5 font-mono">{rule.condition.event_type || '—'}</p>
					</div>
					<div>
						<p class="font-medium text-muted-foreground">Threshold / Window</p>
						<p class="mt-0.5">{rule.condition.threshold} / {rule.condition.window_sec}s</p>
					</div>
				{:else if ruleType === 'sequence'}
					<div class="col-span-2">
						<p class="font-medium text-muted-foreground">Sequence Steps</p>
						<p class="mt-0.5 font-mono">{(rule.condition.sequence_steps ?? []).join(' → ') || '—'}</p>
					</div>
					<div>
						<p class="font-medium text-muted-foreground">Window</p>
						<p class="mt-0.5">{rule.condition.window_sec}s</p>
					</div>
				{:else if ruleType === 'correlation'}
					<div class="col-span-2">
						<p class="font-medium text-muted-foreground">Correlation Event Types</p>
						<p class="mt-0.5 font-mono">{(rule.condition.correlation_event_types ?? []).join(', ') || '—'}</p>
					</div>
					<div>
						<p class="font-medium text-muted-foreground">Threshold / Window</p>
						<p class="mt-0.5">{rule.condition.threshold} / {rule.condition.window_sec}s</p>
					</div>
					<div>
						<p class="font-medium text-muted-foreground">Min Distinct Event Types</p>
						<p class="mt-0.5">{rule.condition.min_distinct_event_types}</p>
					</div>
				{:else}
					<div>
						<p class="font-medium text-muted-foreground">Event Type</p>
						<p class="mt-0.5 font-mono">{rule.condition.event_type || '—'}</p>
					</div>
					<div>
						<p class="font-medium text-muted-foreground">Window / Baseline</p>
						<p class="mt-0.5">{rule.condition.window_sec}s / {rule.condition.baseline_window_sec}s</p>
					</div>
					<div>
						<p class="font-medium text-muted-foreground">Spike Factor</p>
						<p class="mt-0.5">{rule.condition.spike_factor}</p>
					</div>
					<div>
						<p class="font-medium text-muted-foreground">Anomaly Min Count</p>
						<p class="mt-0.5">{rule.condition.anomaly_min_count}</p>
					</div>
				{/if}
				<div class="col-span-2">
					<p class="font-medium text-muted-foreground">Group By</p>
					<p class="mt-0.5 font-mono">{rule.condition.group_by?.join(', ') || '—'}</p>
				</div>
				<div class="col-span-2">
					<p class="font-medium text-muted-foreground">Tags</p>
					<p class="mt-0.5">{rule.tags?.join(', ') || '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Created At</p>
					<div class="mt-0.5"><CompactDate dateString={rule.created_at} /></div>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Updated At</p>
					<div class="mt-0.5"><CompactDate dateString={rule.updated_at} /></div>
				</div>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (viewOpen = false)}>Close</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
