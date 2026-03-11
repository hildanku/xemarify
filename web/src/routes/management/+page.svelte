<script lang='ts'>
    import { Button } from '$lib/components/ui/button/index.js'
    import * as Card from '$lib/components/ui/card/index.js'
    import { Badge } from '$lib/components/ui/badge/index.js'
    import FileTextIcon from '@lucide/svelte/icons/file-text'
    import BellIcon from '@lucide/svelte/icons/bell'
    import ServerIcon from '@lucide/svelte/icons/server'
    import AlertTriangleIcon from '@lucide/svelte/icons/alert-triangle'

    function _getBadgeVariant(
        level: number,
    ): 'default' | 'secondary' | 'destructive' | 'outline' {
        if (level >= 12) return 'destructive'
        if (level >= 8) return 'default'
        if (level >= 4) return 'secondary'
        return 'outline'
    }

    function _getStatusVariant(
        status: string,
    ): 'default' | 'secondary' | 'destructive' | 'outline' {
        if (status === 'new') return 'destructive'
        if (status === 'acknowledged') return 'default'
        if (status === 'resolved') return 'secondary'
        return 'outline'
    }
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
    <div class="flex items-center justify-between">
        <div>
            <h1 class="text-3xl font-bold">Dashboard</h1>
            <p class="text-muted-foreground">
                Security Information and Event Management
            </p>
        </div>
        <Button onclick={() => console.log('halo ges')}>Refresh</Button>
    </div>

    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card.Root>
            <Card.Header
                class="flex flex-row items-center justify-between space-y-0 pb-2"
            >
                <Card.Title class="text-sm font-medium">Total Events</Card.Title
                >
                <FileTextIcon class="h-4 w-4 text-muted-foreground" />
            </Card.Header>
            <Card.Content>
                <div class="text-2xl font-bold">1,913,231</div>
                <p class="text-xs text-muted-foreground">All security events</p>
            </Card.Content>
        </Card.Root>

        <Card.Root>
            <Card.Header
                class="flex flex-row items-center justify-between space-y-0 pb-2"
            >
                <Card.Title class="text-sm font-medium">Total Alerts</Card.Title
                >
                <BellIcon class="h-4 w-4 text-muted-foreground" />
            </Card.Header>
            <Card.Content>
                <div class="text-2xl font-bold">521</div>
                <p class="text-xs text-muted-foreground">
                    Security notifications
                </p>
            </Card.Content>
        </Card.Root>

        <Card.Root>
            <Card.Header
                class="flex flex-row items-center justify-between space-y-0 pb-2"
            >
                <Card.Title class="text-sm font-medium">New Alerts</Card.Title>
                <AlertTriangleIcon class="h-4 w-4 text-destructive" />
            </Card.Header>
            <Card.Content>
                <div class="text-2xl font-bold text-destructive">20</div>
                <p class="text-xs text-muted-foreground">Requires attention</p>
            </Card.Content>
        </Card.Root>

        <!-- Online Agents Card -->
        <Card.Root>
            <Card.Header
                class="flex flex-row items-center justify-between space-y-0 pb-2"
            >
                <Card.Title class="text-sm font-medium"
                    >Online Agents</Card.Title
                >
                <ServerIcon class="h-4 w-4 text-muted-foreground" />
            </Card.Header>
            <Card.Content>
                <div class="text-2xl font-bold">
                    <span class="text-green-600">2</span>
                    <span class="text-muted-foreground">/2</span>
                </div>
                <p class="text-xs text-muted-foreground">Connected sources</p>
            </Card.Content>
        </Card.Root>
    </div>

    <Card.Root>
        <Card.Header>
            <Card.Title>Recent Critical Alerts</Card.Title>
            <Card.Description
                >Latest high-priority security alerts</Card.Description
            >
        </Card.Header>
        <Card.Content>
            <p class="text-center text-muted-foreground py-8">
                No critical alerts found
            </p>

            <div class="space-y-4">
                <div
                    class="flex items-center justify-between border rounded-lg p-4"
                >
                    <div class="flex-1 space-y-1">
                        <div class="flex items-center gap-2">
                            <Badge variant="destructive">Level 12</Badge>
                            <Badge variant="default">New</Badge>
                        </div>
                        <p class="text-sm text-muted-foreground">
                            Alert #20 - Event #1,212,592 Rule: #22...
                        </p>
                        <p class="text-xs text-muted-foreground">
                            11 Maret 2025, 11.40 AM
                        </p>
                    </div>
                </div>
            </div>
        </Card.Content>
    </Card.Root>
</div>
