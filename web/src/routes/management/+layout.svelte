<script lang="ts">
	import AppSidebar from "$lib/components/app-sidebar.svelte";
	import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import { page } from '$app/stores';

	let { children } = $props();

	// Get breadcrumb from current path
	const getBreadcrumbs = (pathname: string) => {
		const paths = pathname.split('/').filter(Boolean);
		const breadcrumbs = [];
		
		for (let i = 0; i < paths.length; i++) {
			const path = '/' + paths.slice(0, i + 1).join('/');
			const title = paths[i].charAt(0).toUpperCase() + paths[i].slice(1);
			breadcrumbs.push({ path, title });
		}
		
		return breadcrumbs;
	};

	const breadcrumbs = $derived(getBreadcrumbs($page.url.pathname));
</script>

<Sidebar.Provider>
	<AppSidebar />
	<Sidebar.Inset>
		<header class="flex h-16 shrink-0 items-center gap-2 border-b bg-background">
			<div class="flex items-center gap-2 px-4">
				<Sidebar.Trigger class="-ms-1" />
				<Separator orientation="vertical" class="me-2 data-[orientation=vertical]:h-4" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						{#each breadcrumbs as crumb, index}
							{#if index > 0}
								<Breadcrumb.Separator />
							{/if}
							<Breadcrumb.Item class={index === 0 ? 'hidden md:block' : ''}>
								{#if index === breadcrumbs.length - 1}
									<Breadcrumb.Page>{crumb.title}</Breadcrumb.Page>
								{:else}
									<Breadcrumb.Link href={crumb.path}>{crumb.title}</Breadcrumb.Link>
								{/if}
							</Breadcrumb.Item>
						{/each}
					</Breadcrumb.List>
				</Breadcrumb.Root>
			</div>
		</header>
		<div class="flex flex-1 flex-col">
			{@render children?.()}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
