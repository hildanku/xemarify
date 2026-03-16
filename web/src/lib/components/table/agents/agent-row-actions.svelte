<script lang="ts">
	import { z } from 'zod'
	import { superForm, defaults } from 'sveltekit-superforms'
	import { zod4Client, zod4 } from 'sveltekit-superforms/adapters'
	import type { Agent } from '$lib/types/api'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import * as Form from '$lib/components/ui/form/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import AgentStatusBadge from './agent-status-badge.svelte'
	import CompactDate from '../../ui/custom/compact-date.svelte'
	import MoreHorizontalIcon from '@lucide/svelte/icons/more-horizontal'

	let {
		agent,
		onDelete,
		onEdit,
	}: {
		agent: Agent
		onDelete: (id: string) => void
		onEdit: (id: string, data: {
			name: string
			hostname?: string
			ip_address?: string
			version?: string
			status: Agent['status']
		}) => void
	} = $props()

	let viewOpen = $state(false)
	let editOpen = $state(false)

	const STATUSES = ['ONLINE', 'OFFLINE'] as const

	const agentSchema = z.object({
		name: z.string().min(1, 'Name is required').max(100),
		hostname: z.string().max(253).optional().default(''),
		ip_address: z.string().max(50).optional().default(''),
		version: z.string().max(50).optional().default(''),
		status: z.enum(STATUSES),
	})

	type AgentFormData = z.infer<typeof agentSchema>

	const form = superForm(defaults(zod4(agentSchema)), {
		validators: zod4Client(agentSchema),
		SPA: true,
		onUpdate({ form: fd }) {
			if (fd.valid) {
				const payload: {
					name: string
					hostname?: string
					ip_address?: string
					version?: string
					status: Agent['status']
				} = {
					name: fd.data.name,
					status: fd.data.status,
				}
				if (fd.data.hostname) payload.hostname = fd.data.hostname
				if (fd.data.ip_address) payload.ip_address = fd.data.ip_address
				if (fd.data.version) payload.version = fd.data.version

				onEdit(agent.id, payload)
				editOpen = false
			}
		},
	})

	const { form: formData, enhance } = form

	// Reset form fields with current agent values when the dialog opens
	$effect(() => {
		if (editOpen) {
			form.reset({
				data: {
					name: agent.name,
					hostname: agent.hostname ?? '',
					ip_address: agent.ip_address ?? '',
					version: agent.version ?? '',
					status: agent.status,
				} satisfies AgentFormData,
			})
		}
	})
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button
				variant="ghost"
				size="sm"
				class="h-8 w-8 p-0"
				aria-label="Open row actions"
				{...props}
			>
				<MoreHorizontalIcon class="h-4 w-4" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-40">
		<DropdownMenu.Item onclick={() => (viewOpen = true)}
			>View details</DropdownMenu.Item
		>
		<DropdownMenu.Item onclick={() => (editOpen = true)}
			>Edit agent</DropdownMenu.Item
		>
		<DropdownMenu.Separator />
		<DropdownMenu.Item
			class="text-destructive focus:text-destructive"
			onclick={() => onDelete(agent.id)}
		>
			Delete agent
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<!-- View details dialog -->
<Dialog.Root bind:open={viewOpen}>
	<Dialog.Content class="max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{agent.name}</Dialog.Title>
			<Dialog.Description>Agent Details</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-2">
			<div class="flex items-center gap-2">
				<AgentStatusBadge status={agent.status} />
			</div>
			<div class="grid grid-cols-2 gap-4 text-sm">
				<div>
					<p class="font-medium text-muted-foreground">Agent ID</p>
					<p class="font-mono text-xs mt-0.5 break-all">{agent.id}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Name</p>
					<p class="mt-0.5">{agent.name}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Hostname</p>
					<p class="mt-0.5">{agent.hostname ?? '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">IP Address</p>
					<p class="font-mono mt-0.5">{agent.ip_address ?? '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Version</p>
					<p class="mt-0.5">{agent.version ?? '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Created At</p>
					<div class="mt-0.5">
						<CompactDate dateString={agent.created_at} />
					</div>
				</div>
				<div class="col-span-2">
					<p class="font-medium text-muted-foreground">Last Seen</p>
					<div class="mt-0.5">
						<CompactDate dateString={agent.last_seen_at} fallback="Never" />
					</div>
				</div>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (viewOpen = false)}>Close</Button
			>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Edit agent dialog (upsert mode) -->
<Dialog.Root bind:open={editOpen}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>Edit Agent</Dialog.Title>
			<Dialog.Description
				>Update agent information. Name is required.</Dialog.Description
			>
		</Dialog.Header>
		<form method="POST" use:enhance class="space-y-4 py-2">
			<Form.Field {form} name="name">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Name</Form.Label>
						<Input
							{...props}
							bind:value={$formData.name}
							placeholder="Agent name"
						/>
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="hostname">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Hostname</Form.Label>
						<Input
							{...props}
							bind:value={$formData.hostname}
							placeholder="e.g. node-asia-01"
						/>
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="ip_address">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>IP Address</Form.Label>
						<Input
							{...props}
							bind:value={$formData.ip_address}
							placeholder="e.g. 192.168.1.1"
						/>
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="version">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Version</Form.Label>
						<Input
							{...props}
							bind:value={$formData.version}
							placeholder="e.g. v2.4.5"
						/>
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="status">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Status</Form.Label>
						<Select.Root type="single" bind:value={$formData.status}>
							<Select.Trigger {...props} class="w-full">
								{$formData.status || 'Select a status'}
							</Select.Trigger>
							<Select.Content>
								{#each STATUSES as status (status)}
									<Select.Item value={status}>{status}</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Dialog.Footer>
				<Button
					variant="outline"
					type="button"
					onclick={() => (editOpen = false)}>Cancel</Button
				>
				<Form.Button>Save changes</Form.Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
