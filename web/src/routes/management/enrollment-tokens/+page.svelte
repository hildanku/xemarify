<script lang="ts">
	import { createMutation } from '@tanstack/svelte-query'
	import { toast } from 'svelte-sonner'
	import { clientFetch } from '$lib/client'
	import { V1_BASE_URL } from '$lib/constant'
	import { Button } from '$lib/components/ui/button/index.js'
	import { Input } from '$lib/components/ui/input/index.js'
	import * as Card from '$lib/components/ui/card/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'
	import { formatDateTime } from '$lib/utils/date'

	type CreateEnrollmentTokenResponse = {
		enrollment_token: string
	}

	let generatedToken = $state('')
	let generatedAt = $state<Date | null>(null)
	let copied = $state(false)

	const managerEndpoint = $derived.by(() => {
		if (typeof window !== 'undefined') {
			return window.location.origin
		}
		return 'http://localhost:8089'
	})

	const onboardingHref = $derived.by(() => {
		const token = encodeURIComponent(generatedToken || 'paste-generated-token-here')
		return `/management/agent-onboarding?token=${token}&endpoint=${encodeURIComponent(managerEndpoint)}`
	})

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
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Enrollment Tokens</h1>
			<p class="text-muted-foreground">
				Generate one-time bootstrap credentials for new agents
			</p>
		</div>
		<Button variant="outline" href={onboardingHref}>Open Onboarding Guide</Button>
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
						Generated at {formatDateTime(generatedAt)}
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
			<pre class="rounded-md border bg-muted p-3 text-xs overflow-x-auto"><code>server:
  endpoint: "{managerEndpoint}"
  insecure: false

enrollment_token: "{generatedToken || 'paste-generated-token-here'}"

agent:
  id: ""
  agent_secret: ""
  name: "web-01"
  hostname: "web-01"
  ip_address: "10.0.0.12"</code></pre>
		</Card.Content>
	</Card.Root>
</div>

<svelte:head>
	<title>Xemarify - Enrollment Tokens</title>
</svelte:head>
