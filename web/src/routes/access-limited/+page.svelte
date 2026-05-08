<script lang="ts">
	import { goto } from '$app/navigation'
	import { resolve } from '$app/paths'
	import { auth, logout } from '$lib/auth/session'
	import { Button } from '$lib/components/ui/button'

	async function handleLogout() {
		try {
			await logout()
		} finally {
			await goto(resolve('/auth/login'), { replaceState: true })
		}
	}
</script>

<svelte:head>
	<title>Xemarify - Access Limited</title>
</svelte:head>

<div class="flex min-h-svh items-center justify-center bg-muted/30 px-6 py-10">
	<div class="w-full max-w-lg rounded-2xl border bg-background p-8 shadow-sm">
		<p class="text-muted-foreground text-sm font-medium">Signed in as {$auth.user?.role ?? 'UNKNOWN'}</p>
		<h1 class="mt-3 text-3xl font-semibold tracking-tight">Frontend access is not available for this role yet.</h1>
		<p class="text-muted-foreground mt-3 text-sm leading-6">
			Your account is authenticated, but the current frontend only exposes management views for
			`MANAGER` and `ANALYST`.
		</p>
		<p class="text-muted-foreground mt-2 text-sm leading-6">
			If `VIEWER` should have its own dashboard or read-only pages, the route policy and backend
			permissions need to be defined first.
		</p>
		<div class="mt-6 flex gap-3">
			<Button onclick={handleLogout}>Log out</Button>
		</div>
	</div>
</div>
