<script lang="ts">
	import { createMutation } from '@tanstack/svelte-query'
	import { toast } from 'svelte-sonner'
	import { clientFetch } from '$lib/client'
	import { V1_BASE_URL } from '$lib/constant'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import * as Card from '$lib/components/ui/card/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'

	type CreateAgentKeyResponse = {
		key: string
	}

	let generatedKey = $state('')
	let generatedAt = $state<Date | null>(null)
	let copied = $state(false)

	const generateKeyMutation = createMutation(() => ({
		mutationFn: () =>
			clientFetch<CreateAgentKeyResponse>(`${V1_BASE_URL}/admin/agent-keys`, {
				method: 'POST',
			}),
		onSuccess: (data) => {
			generatedKey = data.key
			generatedAt = new Date()
			copied = false
			toast.success('Enrollment key generated successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to generate enrollment key: ${error.message}`)
		},
	}))

	async function copyKey() {
		if (!generatedKey) return

		try {
			await navigator.clipboard.writeText(generatedKey)
			copied = true
			toast.success('Enrollment key copied to clipboard')
			setTimeout(() => {
				copied = false
			}, 1500)
		} catch {
			toast.error('Failed to copy key')
		}
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<div>
		<h1 class="text-3xl font-bold tracking-tight">Agent Keys</h1>
		<p class="text-muted-foreground">
			Generate one-time enrollment keys for new agents
		</p>
	</div>

	<Card.Root>
		<Card.Header>
			<Card.Title>Generate Enrollment Key</Card.Title>
			<Card.Description>
				Each generated key can only be used once during agent registration.
			</Card.Description>
		</Card.Header>
		<Card.Content class="space-y-4">
			<div class="flex flex-wrap items-center gap-2">
				<Button
					onclick={() => generateKeyMutation.mutate()}
					disabled={generateKeyMutation.isPending}
				>
					{generateKeyMutation.isPending ? 'Generating…' : 'Generate Key'}
				</Button>

				{#if generatedAt}
					<Badge variant="outline">
						Generated at {generatedAt.toLocaleString()}
					</Badge>
				{/if}
			</div>

			{#if generatedKey}
				<div class="space-y-2">
					<label for="enrollment-key" class="text-sm font-medium">Enrollment Key</label>
					<div class="flex flex-wrap items-center gap-2">
						<Input id="enrollment-key" value={generatedKey} readonly class="font-mono" />
						<Button variant="secondary" onclick={copyKey}>
							{copied ? 'Copied' : 'Copy'}
						</Button>
					</div>
				</div>
			{/if}
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title>Usage</Card.Title>
			<Card.Description>
				Put this value into the agent config under
				<span class="font-mono">agent.agent_key</span> before first run.
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<pre class="rounded-md border bg-muted p-3 text-xs overflow-x-auto"><code>agent:
  id: ""
  key: ""
  agent_key: "{generatedKey || 'paste-generated-key-here'}"</code></pre>
		</Card.Content>
	</Card.Root>
</div>