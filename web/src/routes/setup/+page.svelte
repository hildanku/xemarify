<script lang="ts">
	import { goto } from '$app/navigation'
	import { createMutation } from '@tanstack/svelte-query'
	import ShieldCheck from '@lucide/svelte/icons/shield-check'
	import ShieldAlert from '@lucide/svelte/icons/shield-alert'
	import KeyRound from '@lucide/svelte/icons/key-round'
	import UserRoundCog from '@lucide/svelte/icons/user-round-cog'
	import { toast } from 'svelte-sonner'
	import { writeSession } from '$lib/auth/session'
	import { Button } from '$lib/components/ui/button'
	import { Badge } from '$lib/components/ui/badge'
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle,
	} from '$lib/components/ui/card'
	import {
		Field,
		FieldDescription,
		FieldError,
		FieldGroup,
		FieldLabel,
		FieldSeparator,
	} from '$lib/components/ui/field'
	import { Input } from '$lib/components/ui/input'
	import { Separator } from '$lib/components/ui/separator'
	import { initializeFirstManager } from '$lib/setup/system'

	let username = $state('')
	let email = $state('')
	let password = $state('')
	let confirmPassword = $state('')
	let setupToken = $state('')
	let formError = $state('')

	const initializeMutation = createMutation(() => ({
		mutationFn: async () => {
			if (password !== confirmPassword) {
				throw new Error('Password confirmation does not match')
			}

			return initializeFirstManager({
				username,
				email,
				password,
				setupToken,
			})
		},
		onSuccess: async (tokens) => {
			writeSession(tokens.access_token, tokens.refresh_token)
			toast.success('Initial manager created successfully')
			await goto('/management', { replaceState: true })
		},
		onError: (error) => {
			formError = error instanceof Error ? error.message : 'Setup failed'
			toast.error(formError)
		},
	}))

	function handleSubmit(event: Event) {
		event.preventDefault()
		formError = ''
		initializeMutation.mutate()
	}
</script>

<svelte:head>
	<title>Initial Setup</title>
</svelte:head>

<div class="from-background via-muted/30 to-background min-h-svh bg-gradient-to-br">
	<div class="mx-auto grid min-h-svh max-w-7xl gap-8 px-6 py-10 lg:grid-cols-[1.05fr_0.95fr] lg:px-10">
		<div class="flex flex-col justify-center gap-6">
			<div class="flex items-center gap-3">
				<div class="bg-primary text-primary-foreground flex size-10 items-center justify-center rounded-xl border shadow-sm">
					<ShieldCheck class="size-5" />
				</div>
				<div>
					<p class="text-sm font-semibold tracking-wide">Xemarify</p>
					<p class="text-muted-foreground text-sm">Manager initialization</p>
				</div>
			</div>

			<div class="space-y-4">
				<Badge variant="secondary" class="w-fit">First-run setup</Badge>
				<h1 class="max-w-2xl text-4xl font-semibold tracking-tight text-balance md:text-5xl">
					Initialize the first manager before the instance accepts normal logins.
				</h1>
				<p class="text-muted-foreground max-w-xl text-base leading-7">
					This claims the instance, creates the initial manager account, and immediately closes setup mode for everyone else.
				</p>
			</div>

			<div class="grid gap-4 md:grid-cols-3">
				<Card class="gap-4">
					<CardHeader class="gap-3 px-5">
						<div class="bg-muted flex size-10 items-center justify-center rounded-lg border">
							<UserRoundCog class="text-muted-foreground size-5" />
						</div>
						<CardTitle class="text-base">One-time ownership</CardTitle>
						<CardDescription>
							After the first manager exists, this setup page is no longer reachable.
						</CardDescription>
					</CardHeader>
				</Card>

				<Card class="gap-4">
					<CardHeader class="gap-3 px-5">
						<div class="bg-muted flex size-10 items-center justify-center rounded-lg border">
							<KeyRound class="text-muted-foreground size-5" />
						</div>
						<CardTitle class="text-base">Token protected</CardTitle>
						<CardDescription>
							Use the bootstrap secret configured in `MANAGER_SETUP_TOKEN`.
						</CardDescription>
					</CardHeader>
				</Card>

				<Card class="gap-4">
					<CardHeader class="gap-3 px-5">
						<div class="bg-muted flex size-10 items-center justify-center rounded-lg border">
							<ShieldAlert class="text-muted-foreground size-5" />
						</div>
						<CardTitle class="text-base">Session ready</CardTitle>
						<CardDescription>
							Successful setup signs you in directly and redirects to management.
						</CardDescription>
					</CardHeader>
				</Card>
			</div>
		</div>

		<div class="flex items-center justify-center">
			<Card class="w-full max-w-lg gap-0">
				<CardHeader class="px-6">
					<CardTitle class="text-2xl">Create initial manager</CardTitle>
					<CardDescription>
						This account will own user, rule, agent, and audit administration for the instance.
					</CardDescription>
				</CardHeader>
				<Separator />
				<CardContent class="px-6 pt-6">
					<form class="flex flex-col gap-6" onsubmit={handleSubmit}>
						<FieldGroup>
							<Field>
								<FieldLabel for="username">Username</FieldLabel>
								<Input id="username" bind:value={username} placeholder="manager" required />
							</Field>

							<Field>
								<FieldLabel for="email">Email</FieldLabel>
								<Input
									id="email"
									type="email"
									bind:value={email}
									placeholder="manager@example.com"
									required
								/>
							</Field>

							<Field>
								<FieldLabel for="password">Password</FieldLabel>
								<Input id="password" type="password" bind:value={password} required />
								<FieldDescription>
									Use a strong password with at least 8 characters.
								</FieldDescription>
							</Field>

							<Field>
								<FieldLabel for="confirm-password">Confirm password</FieldLabel>
								<Input id="confirm-password" type="password" bind:value={confirmPassword} required />
							</Field>

							<FieldSeparator>Bootstrap Access</FieldSeparator>

							<Field>
								<FieldLabel for="setup-token">Setup token</FieldLabel>
								<Input id="setup-token" type="password" bind:value={setupToken} required />
								<FieldDescription>
									Paste the value configured in `MANAGER_SETUP_TOKEN`.
								</FieldDescription>
								{#if formError}
									<FieldError>{formError}</FieldError>
								{/if}
							</Field>

							<Field>
								<Button type="submit" class="w-full" disabled={initializeMutation.isPending}>
									{initializeMutation.isPending ? 'Creating manager...' : 'Create initial manager'}
								</Button>
							</Field>
						</FieldGroup>
					</form>
				</CardContent>
			</Card>
		</div>
	</div>
</div>
