<script lang="ts">
	import type { AuditLog } from '$lib/types/api'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import CompactDate from '$lib/components/ui/custom/compact-date.svelte'
	import EyeIcon from '@lucide/svelte/icons/eye'

	let { entry }: { entry: AuditLog } = $props()

	let open = $state(false)

	const metadataText = $derived(
		entry.metadata && Object.keys(entry.metadata).length > 0
			? JSON.stringify(entry.metadata, null, 2)
			: '',
	)
</script>

<Button variant="ghost" size="sm" class="h-8 px-2" onclick={() => (open = true)}>
	<EyeIcon class="mr-2 h-4 w-4" />
	View
</Button>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-2xl">
		<Dialog.Header>
			<Dialog.Title>Audit Log Details</Dialog.Title>
			<Dialog.Description>
				Inspect the selected audit trail entry and its metadata payload.
			</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-4 py-2 text-sm">
			<div class="grid gap-4 md:grid-cols-2">
				<div>
					<p class="font-medium text-muted-foreground">Action</p>
					<p class="mt-0.5 font-mono">{entry.action}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Created At</p>
					<div class="mt-0.5">
						<CompactDate dateString={entry.created_at} />
					</div>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">User</p>
					<p class="mt-0.5 break-all">{entry.user_identifier}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">User ID</p>
					<p class="mt-0.5 break-all font-mono text-xs">{entry.user_id ?? '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Object Type</p>
					<p class="mt-0.5">{entry.object_type ?? '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Object ID</p>
					<p class="mt-0.5 break-all font-mono text-xs">{entry.object_id ?? '—'}</p>
				</div>
			</div>

			<div>
				<p class="font-medium text-muted-foreground">Entry ID</p>
				<p class="mt-0.5 break-all font-mono text-xs">{entry.id}</p>
			</div>

			<div>
				<p class="font-medium text-muted-foreground">Metadata</p>
				{#if metadataText}
					<pre class="mt-1 max-h-80 overflow-auto rounded-md border bg-muted/40 p-3 text-xs leading-5">{metadataText}</pre>
				{:else}
					<p class="mt-0.5 text-muted-foreground">No metadata recorded for this action.</p>
				{/if}
			</div>
		</div>

		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)}>Close</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
