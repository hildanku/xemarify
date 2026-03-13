<script lang="ts">
	import { z } from 'zod'
	import { superForm, defaults } from 'sveltekit-superforms'
	import { zod4Client, zod4 } from 'sveltekit-superforms/adapters'
	import type { Rule, RuleSeverity } from '$lib/types/api'
	import * as Dialog from '$lib/components/ui/dialog/index.js'
	import * as Form from '$lib/components/ui/form/index.js'
	import * as Select from '$lib/components/ui/select/index.js'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import PlusIcon from '@lucide/svelte/icons/plus'

	let {
		mode,
		rule,
		onCreate,
		onEdit,
		isPending = false,
	}: {
		mode: 'create' | 'edit'
		rule?: Rule
		onCreate?: (data: {
			name: string
			description?: string
			level: RuleSeverity
			enabled: boolean
			condition: {
				event_type: string
				group_by: string[]
				threshold: number
				window_sec: number
				severity?: RuleSeverity
			}
			tags?: string[]
		}) => void
		onEdit?: (id: string, data: {
			name: string
			description?: string
			level: RuleSeverity
			enabled: boolean
			condition: {
				event_type: string
				group_by: string[]
				threshold: number
				window_sec: number
				severity?: RuleSeverity
			}
			tags?: string[]
		}) => void
		isPending?: boolean
	} = $props()

	let open = $state(false)
	const levels = ['INFO', 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'] as const

	const schema = z.object({
		name: z.string().min(3).max(120),
		description: z.string().max(500).optional().default(''),
		level: z.enum(levels),
		enabled: z.boolean().default(true),
		event_type: z.string().min(1),
		group_by: z.string().optional().default(''),
		threshold: z.coerce.number().int().min(1),
		window_sec: z.coerce.number().int().min(1),
		condition_severity: z.enum(levels).optional(),
		tags: z.string().optional().default(''),
	})

	const form = superForm(defaults(zod4(schema)), {
		validators: zod4Client(schema),
		SPA: true,
		onUpdate({ form: fd }) {
			if (!fd.valid) return

			const groupBy = fd.data.group_by
				.split(',')
				.map((v) => v.trim())
				.filter(Boolean)
			const tags = fd.data.tags
				.split(',')
				.map((v) => v.trim())
				.filter(Boolean)

			const payload = {
				name: fd.data.name,
				description: fd.data.description || undefined,
				level: fd.data.level,
				enabled: fd.data.enabled,
				condition: {
					event_type: fd.data.event_type,
					group_by: groupBy,
					threshold: fd.data.threshold,
					window_sec: fd.data.window_sec,
					severity: fd.data.condition_severity || undefined,
				},
				tags: tags.length ? tags : undefined,
			}

			if (mode === 'create' && onCreate) {
				onCreate(payload)
				open = false
				form.reset()
			}
			if (mode === 'edit' && onEdit && rule) {
				onEdit(rule.id, payload)
				open = false
			}
		},
	})

	const { form: formData, enhance } = form

	$effect(() => {
		if (open && mode === 'edit' && rule) {
			form.reset({
				data: {
					name: rule.name,
					description: rule.description ?? '',
					level: rule.level,
					enabled: rule.enabled,
					event_type: rule.condition.event_type,
					group_by: (rule.condition.group_by ?? []).join(', '),
					threshold: rule.condition.threshold,
					window_sec: rule.condition.window_sec,
					condition_severity: rule.condition.severity,
					tags: (rule.tags ?? []).join(', '),
				},
			})
		}
	})
</script>

{#if mode === 'create'}
	<Button size="sm" onclick={() => (open = true)}>
		<PlusIcon class="h-4 w-4 mr-2" />
		Add Rule
	</Button>
{:else}
	<Button variant="outline" size="sm" onclick={() => (open = true)}>Edit rule</Button>
{/if}

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-2xl">
		<Dialog.Header>
			<Dialog.Title>{mode === 'create' ? 'Add Rule' : 'Edit Rule'}</Dialog.Title>
			<Dialog.Description>
				{mode === 'create'
					? 'Create a detection rule for threshold-based attacks.'
					: 'Update detection rule configuration.'}
			</Dialog.Description>
		</Dialog.Header>

		<form method="POST" use:enhance class="space-y-4 py-2">
			<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
				<Form.Field {form} name="name">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Rule Name</Form.Label>
							<Input {...props} bind:value={$formData.name} placeholder="SSH Brute Force" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>

				<Form.Field {form} name="level">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Level</Form.Label>
							<Select.Root type="single" bind:value={$formData.level}>
								<Select.Trigger {...props} class="w-full">{$formData.level || 'Select level'}</Select.Trigger>
								<Select.Content>
									{#each levels as l (l)}
										<Select.Item value={l}>{l}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>
			</div>

			<Form.Field {form} name="description">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Description</Form.Label>
						<Input {...props} bind:value={$formData.description} placeholder="Optional description" />
					{/snippet}
				</Form.Control>
				<Form.FieldErrors class="text-xs" />
			</Form.Field>

			<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
				<Form.Field {form} name="event_type">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Event Type</Form.Label>
							<Input {...props} bind:value={$formData.event_type} placeholder="auth_failed" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>

				<Form.Field {form} name="group_by">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Group By</Form.Label>
							<Input {...props} bind:value={$formData.group_by} placeholder="src_ip, hostname" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>
			</div>

			<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
				<Form.Field {form} name="threshold">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Threshold</Form.Label>
							<Input {...props} type="number" bind:value={$formData.threshold} min="1" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>

				<Form.Field {form} name="window_sec">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Window (sec)</Form.Label>
							<Input {...props} type="number" bind:value={$formData.window_sec} min="1" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>

				<Form.Field {form} name="enabled">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Enabled</Form.Label>
							<label class="inline-flex h-10 items-center gap-2 rounded-md border px-3" {...props}>
								<input type="checkbox" class="h-4 w-4" bind:checked={$formData.enabled} />
								<span class="text-sm">{$formData.enabled ? 'Enabled' : 'Disabled'}</span>
							</label>
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>
			</div>

			<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
				<Form.Field {form} name="condition_severity">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Condition Severity</Form.Label>
							<Select.Root type="single" bind:value={$formData.condition_severity}>
								<Select.Trigger {...props} class="w-full">{$formData.condition_severity || 'Optional'}</Select.Trigger>
								<Select.Content>
									{#each levels as l (l)}
										<Select.Item value={l}>{l}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>

				<Form.Field {form} name="tags">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Tags</Form.Label>
							<Input {...props} bind:value={$formData.tags} placeholder="auth, bruteforce" />
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>
			</div>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (open = false)}>Cancel</Button>
				<Form.Button disabled={isPending}>
					{isPending ? (mode === 'create' ? 'Creating…' : 'Saving…') : mode === 'create' ? 'Create rule' : 'Save changes'}
				</Form.Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
