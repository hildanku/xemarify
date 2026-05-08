<script lang="ts">
	import { createMutation } from '@tanstack/svelte-query'
	import { toast } from 'svelte-sonner'
	import { clientFetch } from '$lib/client'
	import { V1_BASE_URL } from '$lib/constant'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import * as Card from '$lib/components/ui/card/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'

	type CreateEnrollmentTokenResponse = {
		enrollment_token: string
	}

	let generatedToken = $state('')
	let generatedAt = $state<Date | null>(null)
	let copied = $state(false)

	const generateTokenMutation = createMutation(() => ({
		mutationFn: () =>
			clientFetch<CreateEnrollmentTokenResponse>(`${V1_BASE_URL}/admin/enrollment-tokens`, {
				method: 'POST',
			}),
		onSuccess: (data) => {
			generatedToken = data.enrollment_token
			generatedAt = new Date()
			copied = false
			toast.success('Enrollment token generated successfully')
		},
		onError: (error: Error) => {
			toast.error(`Failed to generate enrollment token: ${error.message}`)
		},
	}))

	async function copyToken() {
		if (!generatedToken) return

		try {
			await navigator.clipboard.writeText(generatedToken)
			copied = true
			toast.success('Enrollment token copied to clipboard')
			setTimeout(() => {
				copied = false
			}, 1500)
		} catch {
			toast.error('Failed to copy token')
		}
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4 max-w-full">
	<div>
		<h1 class="text-3xl font-bold tracking-tight">Enrollment Tokens</h1>
		<p class="text-muted-foreground">
			Generate one-time bootstrap credentials for new agents
		</p>
	</div>

	<Card.Root>
		<Card.Header>
			<Card.Title>Generate Enrollment Token</Card.Title>
			<Card.Description>
				Each generated token can only be used once during agent enrollment.
			</Card.Description>
		</Card.Header>
		<Card.Content class="space-y-4">
			<div class="flex flex-wrap items-center gap-2">
				<Button
					onclick={() => generateTokenMutation.mutate()}
					disabled={generateTokenMutation.isPending}
				>
					{generateTokenMutation.isPending ? 'Generating…' : 'Generate Token'}
				</Button>

				{#if generatedAt}
					<Badge variant="outline">
						Generated at {generatedAt.toLocaleString()}
					</Badge>
				{/if}
			</div>

			{#if generatedToken}
				<div class="space-y-2">
					<label for="enrollment-token" class="text-sm font-medium">Enrollment Token</label>
					<div class="flex flex-wrap items-center gap-2">
						<Input id="enrollment-token" value={generatedToken} readonly class="font-mono" />
						<Button variant="secondary" onclick={copyToken}>
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
				Put this value into the bootstrap config before the first start.
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<pre class="rounded-md border bg-muted p-3 text-xs overflow-x-auto"><code>manager_url: "https://manager.example.com"
enrollment_token: "{generatedToken || 'paste-generated-token-here'}"

agent:
  name: "web-01"
  hostname: "web-01"
  ip_address: "10.0.0.12"</code></pre>
		</Card.Content>
	</Card.Root>
</div>
