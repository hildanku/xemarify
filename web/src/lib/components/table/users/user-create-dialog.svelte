<script lang="ts">
	import { z } from 'zod'
	import { superForm, defaults } from 'sveltekit-superforms'
	import { zod4Client, zod4 } from 'sveltekit-superforms/adapters'
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
			username: string
			email: string
			role: string
			password: string
			avatar?: string
		}) => void
		isPending?: boolean
	} = $props()

	let open = $state(false)

	const ROLES = ['MANAGER', 'ANALYST', 'VIEWER'] as const

	const createSchema = z.object({
		username: z.string().min(3, 'Minimum 3 characters').max(50),
		email: z.string().email('Invalid email address'),
		role: z.enum(['MANAGER', 'ANALYST', 'VIEWER']),
		password: z.string().min(8, 'Minimum 8 characters'),
		avatar: z.string().max(100).optional().default(''),
	})

	const form = superForm(defaults(zod4(createSchema)), {
		validators: zod4Client(createSchema),
		SPA: true,
		onUpdate({ form: fd }) {
			if (fd.valid) {
				const payload: {
					username: string
					email: string
					role: string
					password: string
					avatar?: string
				} = {
					username: fd.data.username,
					email: fd.data.email,
					role: fd.data.role,
					password: fd.data.password,
				}
				if (fd.data.avatar) payload.avatar = fd.data.avatar
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
	Add User
</Button>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>Add User</Dialog.Title>
			<Dialog.Description>Create a new system user with a role and password.</Dialog.Description>
		</Dialog.Header>
		<form method="POST" use:enhance class="space-y-4 py-2">
			<Form.Field {form} name="username">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Username</Form.Label>
						<Input {...props} bind:value={$formData.username} placeholder="johndoe" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="email">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Email</Form.Label>
						<Input {...props} type="email" bind:value={$formData.email} placeholder="user@example.com" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="role">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Role</Form.Label>
						<Select.Root type="single" bind:value={$formData.role}>
							<Select.Trigger {...props} class="w-full">
								{$formData.role || 'Select a role'}
							</Select.Trigger>
							<Select.Content>
								{#each ROLES as r (r)}
									<Select.Item value={r}>{r}</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="password">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Password</Form.Label>
						<Input {...props} type="password" bind:value={$formData.password} placeholder="Min. 8 characters" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Form.Field {form} name="avatar">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>
							Avatar <span class="text-muted-foreground text-xs">(optional, e.g. initials)</span>
						</Form.Label>
						<Input {...props} bind:value={$formData.avatar} placeholder="e.g. JD" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (open = false)}>Cancel</Button>
				<Form.Button disabled={isPending}>
					{isPending ? 'Creating…' : 'Create user'}
				</Form.Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
