<script lang="ts">
	import { z } from 'zod'
	import { superForm, defaults } from 'sveltekit-superforms'
	import { zod4Client, zod4 } from 'sveltekit-superforms/adapters'
	import type { User } from '$lib/types/api'
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import * as Form from '$lib/components/ui/form/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import UserRoleBadge from './user-role-badge.svelte'
	import CompactDate from '../../ui/custom/compact-date.svelte'
	import MoreHorizontalIcon from '@lucide/svelte/icons/more-horizontal'

	let {
		user,
		onDelete,
		onEdit,
	}: {
		user: User
		onDelete: (id: string) => void
		onEdit: (id: string, data: { username: string; email: string; role: string; avatar?: string }) => void
	} = $props()

	let viewOpen = $state(false)
	let editOpen = $state(false)

	const ROLES = ['MANAGER', 'ANALYST', 'VIEWER'] as const

	const userSchema = z.object({
		username: z.string().min(3, 'Minimum 3 characters').max(50),
		email: z.string().email('Invalid email address'),
		role: z.enum(['MANAGER', 'ANALYST', 'VIEWER']),
		avatar: z.string().max(100).optional().default(''),
	})

	type UserFormData = z.infer<typeof userSchema>

	const form = superForm(defaults(zod4(userSchema)), {
		validators: zod4Client(userSchema),
		SPA: true,
		onUpdate({ form: fd }) {
			if (fd.valid) {
				const payload: { username: string; email: string; role: string; avatar?: string } = {
					username: fd.data.username,
					email: fd.data.email,
					role: fd.data.role,
				}
				if (fd.data.avatar) payload.avatar = fd.data.avatar
				onEdit(user.id, payload)
				editOpen = false
			}
		},
	})

	const { form: formData, enhance } = form

	$effect(() => {
		if (editOpen) {
			form.reset({
				data: {
					username: user.username,
					email: user.email,
					role: user.role,
					avatar: user.avatar ?? '',
				} satisfies UserFormData,
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
		<DropdownMenu.Item onclick={() => (viewOpen = true)}>View details</DropdownMenu.Item>
		<DropdownMenu.Item onclick={() => (editOpen = true)}>Edit user</DropdownMenu.Item>
		<DropdownMenu.Separator />
		<DropdownMenu.Item
			class="text-destructive focus:text-destructive"
			onclick={() => onDelete(user.id)}
		>
			Delete user
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<!-- View details dialog -->
<Dialog.Root bind:open={viewOpen}>
	<Dialog.Content class="max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{user.username}</Dialog.Title>
			<Dialog.Description>User Details</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-2">
			<div>
				<UserRoleBadge role={user.role} />
			</div>
			<div class="grid grid-cols-2 gap-4 text-sm">
				<div>
					<p class="font-medium text-muted-foreground">User ID</p>
					<p class="font-mono text-xs mt-0.5 break-all">{user.id}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Username</p>
					<p class="mt-0.5">{user.username}</p>
				</div>
				<div class="col-span-2">
					<p class="font-medium text-muted-foreground">Email</p>
					<p class="mt-0.5">{user.email}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Avatar</p>
					<p class="mt-0.5">{user.avatar ?? '—'}</p>
				</div>
				<div>
					<p class="font-medium text-muted-foreground">Created At</p>
					<div class="mt-0.5">
						<CompactDate dateString={user.created_at} />
					</div>
				</div>
				{#if user.updated_at}
					<div class="col-span-2">
						<p class="font-medium text-muted-foreground">Last Updated</p>
						<div class="mt-0.5">
							<CompactDate dateString={user.updated_at} />
						</div>
					</div>
				{/if}
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (viewOpen = false)}>Close</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Edit user dialog -->
<Dialog.Root bind:open={editOpen}>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>Edit User</Dialog.Title>
			<Dialog.Description>Update user information.</Dialog.Description>
		</Dialog.Header>
		<form method="POST" use:enhance class="space-y-4 py-2">
			<Form.Field {form} name="username">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Username</Form.Label>
						<Input {...props} bind:value={$formData.username} placeholder="username" />
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

			<Form.Field {form} name="avatar">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Avatar <span class="text-muted-foreground text-xs">(optional)</span></Form.Label>
						<Input {...props} bind:value={$formData.avatar} placeholder="e.g. JD" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (editOpen = false)}>Cancel</Button>
				<Form.Button>Save changes</Form.Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
