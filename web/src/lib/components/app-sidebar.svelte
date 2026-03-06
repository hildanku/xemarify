<script lang="ts" module>
	import LayoutDashboardIcon from "@lucide/svelte/icons/layout-dashboard";
	import FileTextIcon from "@lucide/svelte/icons/file-text";
	import BellIcon from "@lucide/svelte/icons/bell";
	import ShieldAlertIcon from "@lucide/svelte/icons/shield-alert";
	import ServerIcon from "@lucide/svelte/icons/server";
	import BookOpenIcon from "@lucide/svelte/icons/book-open";
	import SettingsIcon from "@lucide/svelte/icons/settings";
	import ShieldCheckIcon from "@lucide/svelte/icons/shield-check";

	const data = {
		user: {
			name: "Admin",
			email: "admin@semarsiem.local",
			avatar: "/avatars/admin.jpg",
		},
		navMain: [
			{
				title: "Dashboard",
				url: "/dashboard",
				icon: LayoutDashboardIcon,
				isActive: true,
			},
			{
				title: "Events",
				url: "/dashboard/events",
				icon: FileTextIcon,
			},
			{
				title: "Alerts",
				url: "/dashboard/alerts",
				icon: BellIcon,
			},
			{
				title: "Detection Rules",
				url: "/dashboard/rules",
				icon: ShieldAlertIcon,
			},
			{
				title: "Agents",
				url: "/dashboard/agents",
				icon: ServerIcon,
			},
		],
		navSecondary: [
			{
				title: "Documentation",
				url: "#",
				icon: BookOpenIcon,
			},
			{
				title: "Settings",
				url: "#",
				icon: SettingsIcon,
			},
		],
	};
</script>

<script lang="ts">
	import NavMain from "./nav-main.svelte";
	import NavProjects from "./nav-projects.svelte";
	import NavSecondary from "./nav-secondary.svelte";
	import NavUser from "./nav-user.svelte";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import CommandIcon from "@lucide/svelte/icons/command";
	import type { ComponentProps } from "svelte";

	let { ref = $bindable(null), ...restProps }: ComponentProps<typeof Sidebar.Root> = $props();
</script>

<Sidebar.Root bind:ref variant="inset" {...restProps}>
	<Sidebar.Header>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton size="lg">
					{#snippet child({ props })}
						<a href="/dashboard" {...props}>
							<div
								class="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg"
							>
								<ShieldCheckIcon class="size-4" />
							</div>
							<div class="grid flex-1 text-start text-sm leading-tight">
								<span class="truncate font-medium">SemarSIEM</span>
								<span class="truncate text-xs">Security Monitoring</span>
							</div>
						</a>
					{/snippet}
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Header>
	<Sidebar.Content>
		<NavMain items={data.navMain} />
		<NavSecondary items={data.navSecondary} class="mt-auto" />
	</Sidebar.Content>
	<Sidebar.Footer>
		<NavUser user={data.user} />
	</Sidebar.Footer>
</Sidebar.Root>
