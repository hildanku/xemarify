<script lang="ts">
	import '../app.css'
	import favicon from '$lib/assets/favicon.svg'
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query'
	import { browser } from '$app/environment'

	let { children } = $props()
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				enabled: browser,
			}
		}
	})
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<QueryClientProvider client={queryClient}>
	{@render children?.()}
</QueryClientProvider>