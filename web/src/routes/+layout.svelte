<script lang="ts">
	import '../app.css'
	import favicon from '$lib/assets/favicon.svg'
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query'
	import { browser } from '$app/environment'
	import { Toaster } from '$lib/components/ui/sonner'

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
	<title>Xemarify</title>
</svelte:head>

<QueryClientProvider client={queryClient}>
	{@render children?.()}
	<Toaster richColors position="bottom-right" />
</QueryClientProvider>