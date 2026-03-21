<script lang='ts'>
    import type { UserRole } from '$lib/types/api'
    import LayoutmanagementIcon from '@lucide/svelte/icons/layout-dashboard'
    import FileTextIcon from '@lucide/svelte/icons/file-text'
    import BellIcon from '@lucide/svelte/icons/bell'
    import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert'
    import ServerIcon from '@lucide/svelte/icons/server'
    import BookOpenIcon from '@lucide/svelte/icons/book-open'
    import SettingsIcon from '@lucide/svelte/icons/settings'
    import ShieldCheckIcon from '@lucide/svelte/icons/shield-check'
    import { auth } from '$lib/auth/session'
    import NavMain from './nav-main.svelte'
    // import NavProjects from './nav-projects.svelte'
    import NavSecondary from './nav-secondary.svelte'
    import NavUser from './nav-user.svelte'
    import * as Sidebar from '$lib/components/ui/sidebar/index.js'
    // import CommandIcon from '@lucide/svelte/icons/command'
    import type { ComponentProps } from 'svelte'
    let {
        ref = $bindable(null),
        ...restProps
    }: ComponentProps<typeof Sidebar.Root> = $props()

    function getNavMain(role?: UserRole) {
        const items = [
            {
                title: 'management',
                url: '/management',
                icon: LayoutmanagementIcon,
                isActive: true,
            },
            {
                title: 'Events',
                url: '/management/events',
                icon: FileTextIcon,
            },
            {
                title: 'Alerts',
                url: '/management/alerts',
                icon: BellIcon,
            },
        ]

        if (role === 'MANAGER') {
            items.push(
                {
                    title: 'Detection Rules',
                    url: '/management/rules',
                    icon: ShieldAlertIcon,
                },
                {
                    title: 'Agents',
                    url: '/management/agents',
                    icon: ServerIcon,
                },
            )
        }

        return items
    }

    const navSecondary = [
        {
            title: 'Documentation',
            url: '#',
            icon: BookOpenIcon,
        },
        {
            title: 'Settings',
            url: '#',
            icon: SettingsIcon,
        },
    ]

    const user = $derived({
        name: $auth.user?.username ?? 'Unknown User',
        email: $auth.user?.email ?? $auth.user?.role ?? '',
        avatar: '/avatars/admin.jpg',
    })

    const navMain = $derived(getNavMain($auth.user?.role))
</script>

<Sidebar.Root bind:ref variant='inset' {...restProps}>
    <Sidebar.Header>
        <Sidebar.Menu>
            <Sidebar.MenuItem>
                <Sidebar.MenuButton size="lg">
                    {#snippet child({ props })}
                        <a href="/management" {...props}>
                            <div
                                class="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg"
                            >
                                <ShieldCheckIcon class="size-4" />
                            </div>
                            <div
                                class="grid flex-1 text-start text-sm leading-tight"
                            >
                                <span class="truncate font-medium"
                                    >Xemarify</span
                                >
                                <span class="truncate text-xs"
                                    >Security Monitoring</span
                                >
                            </div>
                        </a>
                    {/snippet}
                </Sidebar.MenuButton>
            </Sidebar.MenuItem>
        </Sidebar.Menu>
    </Sidebar.Header>
    <Sidebar.Content>
        <NavMain items={navMain} />
        <NavSecondary items={navSecondary} class="mt-auto" />
    </Sidebar.Content>
    <Sidebar.Footer>
        <NavUser user={user} />
    </Sidebar.Footer>
</Sidebar.Root>
