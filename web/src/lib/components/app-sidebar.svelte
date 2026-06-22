<script lang='ts'>
    import type { UserRole } from '$lib/types/api'
    import LayoutmanagementIcon from '@lucide/svelte/icons/layout-dashboard'
    import FileTextIcon from '@lucide/svelte/icons/file-text'
    import BellIcon from '@lucide/svelte/icons/bell'
    import ClipboardListIcon from '@lucide/svelte/icons/clipboard-list'
    import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert'
    import ServerIcon from '@lucide/svelte/icons/server'
    import BookOpenIcon from '@lucide/svelte/icons/book-open'
    import SettingsIcon from '@lucide/svelte/icons/settings'
    import UserIcon from '@lucide/svelte/icons/user'
    import { mode } from 'mode-watcher'
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
                title: 'Dashboard',
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
            {
                title: 'Audit Logs',
                url: '/management/audit-logs',
                icon: ClipboardListIcon,
            },
        ]

        if (role === 'VIEWER') {
            items.push({
                title: 'Agents',
                url: '/management/agents',
                icon: ServerIcon,
            })
        }

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
                {
                    title: 'Enrollment Tokens',
                    url: '/management/enrollment-tokens',
                    icon: ServerIcon,
                },
                {
                    title: 'Users',
                    url: '/management/users',
                    icon: UserIcon,
                },
            )
        }

        return items
    }

    const navSecondary = [
        {
            title: 'Documentation',
            url: '/management/agent-onboarding',
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
                            <img
                                src={mode.current === 'dark' ? '/assets/logo_for_dark.svg' : '/assets/logo_for_light.svg'}
                                alt="Xemarify"
                                width="200"
                                height="48"
                            />
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
