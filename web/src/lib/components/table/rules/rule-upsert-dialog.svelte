<script lang="ts">
	import { z } from 'zod'
	import { superForm, defaults } from 'sveltekit-superforms'
	import { zod4Client, zod4 } from 'sveltekit-superforms/adapters'
	import type { Rule, RuleSeverity, CreateRuleRequest, UpdateRuleRequest } from '$lib/types/api'
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
		open = $bindable(false),
		hideTrigger = false,
		isPending = false,
	}: {
		mode: 'create' | 'edit'
		rule?: Rule
		onCreate?: (data: CreateRuleRequest) => void
		onEdit?: (id: string, data: UpdateRuleRequest) => void
		open?: boolean
		hideTrigger?: boolean
		isPending?: boolean
	} = $props()
	const levels = ['INFO', 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'] as const
	const ruleTypes = ['threshold', 'sequence', 'correlation', 'anomaly'] as const

	type RuleType = (typeof ruleTypes)[number]

	function parseCsv(input: string | undefined): string[] {
		if (!input) return []
		return input
			.split(',')
			.map((value) => value.trim())
			.filter(Boolean)
	}

	function toOptionalNumber(value: unknown): number | undefined {
		if (value === '' || value === null || value === undefined) return undefined
		if (typeof value === 'number') return Number.isNaN(value) ? undefined : value
		const parsed = Number(value)
		return Number.isNaN(parsed) ? undefined : parsed
	}

	const schema = z.object({
		name: z.string().min(3).max(120),
		description: z.string().max(500).optional().default(''),
		level: z.enum(levels),
		enabled: z.boolean().default(true),
		detection_type: z.enum(ruleTypes).default('threshold'),
		event_type: z.string().optional().default(''),
		group_by: z.string().optional().default(''),
		threshold: z.preprocess(toOptionalNumber, z.number().int().min(1).optional()),
		window_sec: z.preprocess(toOptionalNumber, z.number().int().min(1).optional()),
		sequence_steps_text: z.string().optional().default(''),
		correlation_event_types_text: z.string().optional().default(''),
		min_distinct_event_types: z.preprocess(toOptionalNumber, z.number().int().min(1).optional()),
		baseline_window_sec: z.preprocess(toOptionalNumber, z.number().int().min(1).optional()),
		spike_factor: z.preprocess(toOptionalNumber, z.number().gt(1).optional()),
		anomaly_min_count: z.preprocess(toOptionalNumber, z.number().int().min(1).optional()),
		condition_severity: z.enum(levels).optional(),
		tags: z.string().optional().default(''),
	}).superRefine((data, ctx) => {
		const type = data.detection_type
		const eventType = data.event_type.trim()
		const sequenceSteps = parseCsv(data.sequence_steps_text)
		const correlationTypes = parseCsv(data.correlation_event_types_text)

		if (!data.window_sec) {
			ctx.addIssue({ code: 'custom', path: ['window_sec'], message: 'Window is required' })
		}

		if (type === 'threshold') {
			if (!eventType) {
				ctx.addIssue({ code: 'custom', path: ['event_type'], message: 'Event type is required' })
			}
			if (!data.threshold) {
				ctx.addIssue({ code: 'custom', path: ['threshold'], message: 'Threshold is required' })
			}
		}

		if (type === 'sequence' && sequenceSteps.length < 2) {
			ctx.addIssue({
				code: 'custom',
				path: ['sequence_steps_text'],
				message: 'Sequence steps must contain at least 2 event types',
			})
		}

		if (type === 'correlation') {
			if (correlationTypes.length < 2) {
				ctx.addIssue({
					code: 'custom',
					path: ['correlation_event_types_text'],
					message: 'Correlation event types must contain at least 2 event types',
				})
			}
			if (!data.threshold) {
				ctx.addIssue({ code: 'custom', path: ['threshold'], message: 'Threshold is required' })
			}
			if (!data.min_distinct_event_types) {
				ctx.addIssue({
					code: 'custom',
					path: ['min_distinct_event_types'],
					message: 'Min distinct event types is required',
				})
			}
			if (
				data.min_distinct_event_types &&
				correlationTypes.length > 0 &&
				data.min_distinct_event_types > correlationTypes.length
			) {
				ctx.addIssue({
					code: 'custom',
					path: ['min_distinct_event_types'],
					message: 'Min distinct event types cannot exceed total correlation event types',
				})
			}
		}

		if (type === 'anomaly') {
			if (!eventType) {
				ctx.addIssue({ code: 'custom', path: ['event_type'], message: 'Event type is required' })
			}
			if (!data.baseline_window_sec) {
				ctx.addIssue({
					code: 'custom',
					path: ['baseline_window_sec'],
					message: 'Baseline window is required',
				})
			}
			if (!data.spike_factor) {
				ctx.addIssue({ code: 'custom', path: ['spike_factor'], message: 'Spike factor is required' })
			}
			if (!data.anomaly_min_count) {
				ctx.addIssue({
					code: 'custom',
					path: ['anomaly_min_count'],
					message: 'Anomaly min count is required',
				})
			}
		}
	})

	const form = superForm(defaults(zod4(schema)), {
		validators: zod4Client(schema),
		SPA: true,
		onUpdate({ form: fd }) {
			if (!fd.valid) return

			const ruleType = fd.data.detection_type as RuleType
			const groupBy = parseCsv(fd.data.group_by)
			const tags = parseCsv(fd.data.tags)
			const sequenceSteps = parseCsv(fd.data.sequence_steps_text)
			const correlationEventTypes = parseCsv(fd.data.correlation_event_types_text)

			const condition: CreateRuleRequest['condition'] = {
				type: ruleType,
				group_by: groupBy,
				window_sec: fd.data.window_sec,
				severity: fd.data.condition_severity || undefined,
			}

			if (ruleType === 'threshold') {
				condition.event_type = fd.data.event_type.trim()
				condition.threshold = fd.data.threshold
			}

			if (ruleType === 'sequence') {
				condition.sequence_steps = sequenceSteps
			}

			if (ruleType === 'correlation') {
				condition.threshold = fd.data.threshold
				condition.correlation_event_types = correlationEventTypes
				condition.min_distinct_event_types = fd.data.min_distinct_event_types
			}

			if (ruleType === 'anomaly') {
				condition.event_type = fd.data.event_type.trim()
				condition.baseline_window_sec = fd.data.baseline_window_sec
				condition.spike_factor = fd.data.spike_factor
				condition.anomaly_min_count = fd.data.anomaly_min_count
			}

			const payload: CreateRuleRequest = {
				name: fd.data.name,
				description: fd.data.description || undefined,
				level: fd.data.level,
				enabled: fd.data.enabled,
				condition,
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
			const conditionType = (rule.condition.type ?? 'threshold') as RuleType
			form.reset({
				data: {
					name: rule.name,
					description: rule.description ?? '',
					level: rule.level,
					enabled: rule.enabled,
					detection_type: conditionType,
					event_type: rule.condition.event_type ?? '',
					group_by: (rule.condition.group_by ?? []).join(', '),
					threshold: rule.condition.threshold,
					window_sec: rule.condition.window_sec,
					sequence_steps_text: (rule.condition.sequence_steps ?? []).join(', '),
					correlation_event_types_text: (rule.condition.correlation_event_types ?? []).join(', '),
					min_distinct_event_types: rule.condition.min_distinct_event_types,
					baseline_window_sec: rule.condition.baseline_window_sec,
					spike_factor: rule.condition.spike_factor,
					anomaly_min_count: rule.condition.anomaly_min_count,
					condition_severity: rule.condition.severity,
					tags: (rule.tags ?? []).join(', '),
				},
			})
		}
	})
</script>

{#if !hideTrigger}
	{#if mode === 'create'}
		<Button size="sm" onclick={() => (open = true)}>
			<PlusIcon class="h-4 w-4 mr-2" />
			Add Rule
		</Button>
	{:else}
		<Button size="sm" onclick={() => (open = true)}>Edit rule</Button>
	{/if}
{/if}

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-2xl">
		<Dialog.Header>
			<Dialog.Title>{mode === 'create' ? 'Add Rule' : 'Edit Rule'}</Dialog.Title>
			<Dialog.Description>
				{mode === 'create'
					? 'Create a detection rule for threshold, sequence, correlation, or anomaly behavior.'
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
				<Form.Field {form} name="detection_type">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Detection Type</Form.Label>
							<Select.Root type="single" bind:value={$formData.detection_type}>
								<Select.Trigger {...props} class="w-full">{$formData.detection_type || 'Select type'}</Select.Trigger>
								<Select.Content>
									{#each ruleTypes as t (t)}
										<Select.Item value={t}>{t}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						{/snippet}
					</Form.Control>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>

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

			<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
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

				{#if $formData.detection_type === 'threshold' || $formData.detection_type === 'correlation'}
					<Form.Field {form} name="threshold">
						<Form.Control>
							{#snippet children({ props })}
								<Form.Label>Threshold</Form.Label>
								<Input {...props} type="number" bind:value={$formData.threshold} min="1" />
							{/snippet}
						</Form.Control>
						<Form.FieldErrors class="text-xs" />
					</Form.Field>
				{/if}
			</div>

			{#if $formData.detection_type === 'sequence'}
				<Form.Field {form} name="sequence_steps_text">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Sequence Steps</Form.Label>
							<Input {...props} bind:value={$formData.sequence_steps_text} placeholder="auth_failed, auth_failed, auth_success" />
						{/snippet}
					</Form.Control>
					<Form.Description class="text-xs">Use comma-separated event types in order.</Form.Description>
					<Form.FieldErrors class="text-xs" />
				</Form.Field>
			{/if}

			{#if $formData.detection_type === 'correlation'}
				<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
					<Form.Field {form} name="correlation_event_types_text">
						<Form.Control>
							{#snippet children({ props })}
								<Form.Label>Correlation Event Types</Form.Label>
								<Input {...props} bind:value={$formData.correlation_event_types_text} placeholder="network_scan, auth_failed" />
							{/snippet}
						</Form.Control>
						<Form.FieldErrors class="text-xs" />
					</Form.Field>

					<Form.Field {form} name="min_distinct_event_types">
						<Form.Control>
							{#snippet children({ props })}
								<Form.Label>Min Distinct Event Types</Form.Label>
								<Input {...props} type="number" bind:value={$formData.min_distinct_event_types} min="1" />
							{/snippet}
						</Form.Control>
						<Form.FieldErrors class="text-xs" />
					</Form.Field>
				</div>
			{/if}

			{#if $formData.detection_type === 'anomaly'}
				<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
					<Form.Field {form} name="baseline_window_sec">
						<Form.Control>
							{#snippet children({ props })}
								<Form.Label>Baseline Window (sec)</Form.Label>
								<Input {...props} type="number" bind:value={$formData.baseline_window_sec} min="1" />
							{/snippet}
						</Form.Control>
						<Form.FieldErrors class="text-xs" />
					</Form.Field>

					<Form.Field {form} name="spike_factor">
						<Form.Control>
							{#snippet children({ props })}
								<Form.Label>Spike Factor</Form.Label>
								<Input {...props} type="number" step="0.1" bind:value={$formData.spike_factor} min="1.1" />
							{/snippet}
						</Form.Control>
						<Form.FieldErrors class="text-xs" />
					</Form.Field>

					<Form.Field {form} name="anomaly_min_count">
						<Form.Control>
							{#snippet children({ props })}
								<Form.Label>Anomaly Min Count</Form.Label>
								<Input {...props} type="number" bind:value={$formData.anomaly_min_count} min="1" />
							{/snippet}
						</Form.Control>
						<Form.FieldErrors class="text-xs" />
					</Form.Field>
				</div>
			{/if}

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
