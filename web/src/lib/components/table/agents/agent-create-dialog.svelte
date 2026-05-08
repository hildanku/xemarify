<script lang="ts">
	import { z } from 'zod'
	import { superForm, defaults } from 'sveltekit-superforms'
	import { zod4Client, zod4 } from 'sveltekit-superforms/adapters'
	import type { AgentStatus } from '$lib/types/api'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import * as Form from '$lib/components/ui/form/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import PlusIcon from '@lucide/svelte/icons/plus'

	let {
		onCreate,
		isPending = false,
	}: {
		onCreate: (data: {
			name: string
			hostname?: string
			ip_address?: string
			version?: string
			status?: AgentStatus
			agent_secret?: string
		}) => void
		isPending?: boolean
	} = $props()

	let open = $state(false)

	const STATUSES = ['ONLINE', 'OFFLINE'] as const

	const createSchema = z.object({
		name: z.string().min(1, 'Name is required').max(100),
		hostname: z.string().max(253).optional().default(''),
		ip_address: z.string().max(50).optional().default(''),
		version: z.string().max(50).optional().default(''),
		status: z.enum(STATUSES).default('OFFLINE'),
		agent_secret: z.string().max(255).optional().default(''),
	})

	const form = superForm(defaults(zod4(createSchema)), {
		validators: zod4Client(createSchema),
		SPA: true,
		onUpdate({ form: fd }) {
			if (fd.valid) {
				const payload: {
					name: string
					hostname?: string
					ip_address?: string
					version?: string
					status?: AgentStatus
					agent_secret?: string
				} = {
					name: fd.data.name,
					status: fd.data.status,
				}

				if (fd.data.hostname) payload.hostname = fd.data.hostname
				if (fd.data.ip_address) payload.ip_address = fd.data.ip_address
				if (fd.data.version) payload.version = fd.data.version
				if (fd.data.agent_secret) payload.agent_secret = fd.data.agent_secret

				onCreate(payload)
				open = false
				form.reset()
			}
		},
	})

	const { form: formData, enhance } = form
</script>

<Button size="sm" onclick={() => (open = true)}>
	<PlusIcon class="h-4 w-4 mr-2" />
	Add Agent
</Button>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>Add Agent</Dialog.Title>
			<Dialog.Description>Create a new agent record for monitoring.</Dialog.Description>
		</Dialog.Header>
		<form method="POST" use:enhance class="space-y-4 py-2">
			<Form.Field {form} name="name">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Name</Form.Label>
						<Input {...props} bind:value={$formData.name} placeholder="Agent name" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="hostname">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Hostname</Form.Label>
						<Input {...props} bind:value={$formData.hostname} placeholder="e.g. node-asia-01" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
				<Form.Field {form} name="ip_address">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>IP Address</Form.Label>
							<Input {...props} bind:value={$formData.ip_address} placeholder="192.168.1.10" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>

				<Form.Field {form} name="version">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Version</Form.Label>
							<Input {...props} bind:value={$formData.version} placeholder="e.g. v1.0.0" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>
			</div>

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

			<Form.Field {form} name="agent_secret">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>
							Agent Secret <span class="text-muted-foreground text-xs">(optional)</span>
						</Form.Label>
						<Input
							{...props}
							bind:value={$formData.agent_secret}
							placeholder="Optional runtime secret"
						/>
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (open = false)}>Cancel</Button>
				<Form.Button disabled={isPending}>
					{isPending ? 'Creating…' : 'Create agent'}
				</Form.Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
