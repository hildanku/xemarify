<script lang="ts">
	import { goto } from '$app/navigation'
	import { onMount } from 'svelte'
	import { bootstrapSession } from '$lib/auth/session'

	onMount(async () => {
		const state = await bootstrapSession()
		await goto(
			state.status === 'authenticated' ? '/management' : '/auth/login',
			{ replaceState: true },
		)
	})
</script>

<svelte:head>
	<title>Redirecting...</title>
</svelte:head>

<div class="redirect-state">
	<h1>Redirecting...</h1>
	<p>Checking your session and sending you to the right page.</p>
</div>

<style>
	.redirect-state {
		min-height: 100vh;
		display: grid;
		place-content: center;
		gap: 0.75rem;
		padding: 2rem;
		text-align: center;
	}

	h1 {
		margin: 0;
		font-size: clamp(1.75rem, 4vw, 2.5rem);
	}

	p {
		margin: 0;
		color: #4b5563;
	}
</style>
