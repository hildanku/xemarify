<script lang="ts">
	import type { Rule, UpdateRuleRequest } from '$lib/types/api'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import RuleLevelBadge from './rule-level-badge.svelte'
	import RuleUpsertDialog from './rule-upsert-dialog.svelte'
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
		<DropdownMenu.Item>
			<div class="w-full">
				<RuleUpsertDialog mode="edit" {rule} onEdit={onEdit} />
			</div>
		</DropdownMenu.Item>
		<DropdownMenu.Separator />
		<DropdownMenu.Item class="text-destructive focus:text-destructive" onclick={() => onDelete(rule.id)}>
			Delete rule
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<Dialog.Root bind:open={viewOpen}>
	<Dialog.Content class="max-w-2xl">
		<Dialog.Header>
			<Dialog.Title>{rule.name}</Dialog.Title>
			<Dialog.Description>Detection Rule Details</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-2 text-sm">
			<div class="flex items-center gap-2">
				<RuleLevelBadge level={rule.level} />
				<span class="text-muted-foreground">{rule.enabled ? 'Enabled' : 'Disabled'}</span>
			</div>
			<div class="grid grid-cols-2 gap-4">
				<div class="col-span-2">
					<p class="font-medium text-muted-foreground">Description</p>
					<p class="mt-0.5">{rule.description || '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Event Type</p>
					<p class="mt-0.5 font-mono">{rule.condition.event_type}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Threshold / Window</p>
					<p class="mt-0.5">{rule.condition.threshold} / {rule.condition.window_sec}s</p>
				</div>
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
