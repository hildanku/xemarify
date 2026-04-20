<script lang="ts">
	import { goto } from '$app/navigation'
	import { createMutation } from '@tanstack/svelte-query'
	import { z } from 'zod'
	import { superForm, defaults } from 'sveltekit-superforms'
	import { zod4Client, zod4 } from 'sveltekit-superforms/adapters'
	import ShieldAlert from '@lucide/svelte/icons/shield-alert'
	import KeyRound from '@lucide/svelte/icons/key-round'
	import UserRoundCog from '@lucide/svelte/icons/user-round-cog'
	import ArrowRight from '@lucide/svelte/icons/arrow-right'
	import Lock from '@lucide/svelte/icons/lock'
	import { toast } from 'svelte-sonner'
	import { writeSession } from '$lib/auth/session'
	import { Button } from '$lib/components/ui/button'
	import { Badge } from '$lib/components/ui/badge'
	import {
		Field,
		FieldDescription,
		FieldError,
		FieldLabel,
	} from '$lib/components/ui/field'
	import { Input } from '$lib/components/ui/input'
	import { Separator } from '$lib/components/ui/separator'
	import { initializeFirstManager } from '$lib/setup/system'

	let formError = $state('')

	const setupSchema = z
		.object({
			username: z.string().trim().min(3, 'Username must be at least 3 characters'),
			email: z.string().trim().email('Please enter a valid email address'),
			password: z.string().min(8, 'Password must be at least 8 characters'),
			confirmPassword: z.string().min(1, 'Please confirm your password'),
			setupToken: z.string().trim().min(1, 'Setup token is required'),
		})
		.refine((value) => value.password === value.confirmPassword, {
			path: ['confirmPassword'],
			message: 'Password confirmation does not match',
		})

	type SetupFormData = z.infer<typeof setupSchema>

	const initializeMutation = createMutation(() => ({
		mutationFn: async (input: SetupFormData) => {
			return initializeFirstManager({
				username: input.username,
				email: input.email,
				password: input.password,
				setupToken: input.setupToken,
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

	const form = superForm(defaults(zod4(setupSchema)), {
		validators: zod4Client(setupSchema),
		SPA: true,
		onUpdate({ form: fd }) {
			if (fd.valid) {
				formError = ''
				initializeMutation.mutate(fd.data)
			}
		},
	})

	const { form: formData, errors, enhance } = form

	const infoCards = [
		{
			icon: UserRoundCog,
			title: 'One time only',
			description: 'This page closes after the first manager is created.',
		},
		{
			icon: KeyRound,
			title: 'Token required',
			description: 'Use your setup token from environment config.',
		},
		{
			icon: ShieldAlert,
			title: 'Auto sign in',
			description: "You're signed in and redirected upon success.",
		},
	]
</script>

<svelte:head>
	<title>Xemarify - Setup</title>
</svelte:head>

<div class="bg-muted/40 min-h-svh">
	<div class="mx-auto grid min-h-svh max-w-6xl lg:grid-cols-2">
		<div class="border-border/50 flex flex-col justify-center gap-8 border-r bg-white px-10 py-12">
			<div class="flex items-center gap-3">
				<div>
					<p class="font-mono text-sm font-semibold tracking-tight">Xemarify</p>
					<p class="text-muted-foreground text-xs">Setup manager</p>
				</div>
			</div>

			<div class="space-y-3">
				<Badge variant="outline" class="text-muted-foreground rounded-full text-xs font-normal">
					First-run setup
				</Badge>
				<h1 class="text-3xl font-semibold tracking-tight">
					Create the first<br />manager account.
				</h1>
				<p class="text-muted-foreground text-sm leading-relaxed">
					Complete this once to unlock access to the dashboard.
				</p>
			</div>

			<div class="space-y-2">
				{#each infoCards as card}
					<div class="border-border/50 flex items-start gap-3 rounded-lg border bg-white p-3.5">
						<div class="bg-muted border-border/40 mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md border">
							<card.icon class="text-muted-foreground size-3.5" />
						</div>
						<div>
							<p class="text-sm font-medium">{card.title}</p>
							<p class="text-muted-foreground mt-0.5 text-xs leading-relaxed">{card.description}</p>
						</div>
					</div>
				{/each}
			</div>
		</div>

		<div class="flex items-center justify-center bg-white px-10 py-12">
			<div class="w-full max-w-md space-y-6">
				<div>
					<h2 class="text-xl font-semibold tracking-tight">Create manager</h2>
					<p class="text-muted-foreground mt-1 text-sm">Use this account to manage the system.</p>
				</div>

				<form method="POST" use:enhance class="space-y-5">
					<div class="grid grid-cols-2 gap-3">
						<Field>
							<FieldLabel class="text-xs uppercase tracking-wide" for="username">Username</FieldLabel>
							<Input
								id="username"
								bind:value={$formData.username}
								placeholder="manager"
								class="h-9 text-sm"
							/>
							{#if $errors.username}
								<FieldError class="text-xs">{$errors.username[0]}</FieldError>
							{/if}
						</Field>

						<Field>
							<FieldLabel class="text-xs uppercase tracking-wide" for="email">Email</FieldLabel>
							<Input
								id="email"
								type="email"
								bind:value={$formData.email}
								placeholder="you@example.com"
								class="h-9 text-sm"
							/>
							{#if $errors.email}
								<FieldError class="text-xs">{$errors.email[0]}</FieldError>
							{/if}
						</Field>
					</div>

					<div class="grid grid-cols-2 gap-3">
						<Field>
							<FieldLabel class="text-xs uppercase tracking-wide" for="password">Password</FieldLabel>
							<Input
								id="password"
								type="password"
								bind:value={$formData.password}
								placeholder="••••••••"
								class="h-9 text-sm"
							/>
							<FieldDescription class="text-xs">Min. 8 characters.</FieldDescription>
							{#if $errors.password}
								<FieldError class="text-xs">{$errors.password[0]}</FieldError>
							{/if}
						</Field>

						<Field>
							<FieldLabel class="text-xs uppercase tracking-wide" for="confirm-password">Confirm</FieldLabel>
							<Input
								id="confirm-password"
								type="password"
								bind:value={$formData.confirmPassword}
								placeholder="••••••••"
								class="h-9 text-sm"
							/>
							{#if $errors.confirmPassword}
								<FieldError class="text-xs">{$errors.confirmPassword[0]}</FieldError>
							{/if}
						</Field>
					</div>

					<div class="flex items-center gap-3">
						<Separator class="flex-1" />
						<span class="text-muted-foreground text-[10px] uppercase tracking-widest">
							Bootstrap access
						</span>
						<Separator class="flex-1" />
					</div>

					<Field>
						<FieldLabel class="text-xs uppercase tracking-wide" for="setup-token">Setup token</FieldLabel>
						<div class="relative">
							<Lock class="text-muted-foreground absolute top-1/2 left-3 size-3.5 -translate-y-1/2" />
							<Input
								id="setup-token"
								type="password"
								bind:value={$formData.setupToken}
								placeholder="Paste your setup token"
								class="h-9 pl-9 text-sm"
							/>
						</div>
						<FieldDescription class="text-xs">Found in your environment configuration.</FieldDescription>
						{#if $errors.setupToken}
							<FieldError class="text-xs">{$errors.setupToken[0]}</FieldError>
						{/if}
						{#if formError}
							<FieldError class="text-xs">{formError}</FieldError>
						{/if}
					</Field>

					<Button
						type="submit"
						class="h-9 w-full text-sm"
						disabled={initializeMutation.isPending}
					>
						{#if initializeMutation.isPending}
							Creating manager...
						{:else}
							Create initial manager
							<ArrowRight class="ml-1.5 size-3.5" />
						{/if}
					</Button>
				</form>
			</div>
		</div>
	</div>
</div>