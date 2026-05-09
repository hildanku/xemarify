<script lang="ts">
	import { page } from '$app/state'
	import { BASE_URL } from '$lib/constant'
	import { Button } from '$lib/components/ui/button/index.js'
	import * as Card from '$lib/components/ui/card/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'

	const managerEndpoint = $derived.by(() => {
		const endpoint = page.url.searchParams.get('endpoint')?.trim()
		return endpoint || BASE_URL || 'https://manager.example.com'
	})

	const enrollmentToken = $derived.by(() => {
		const token = page.url.searchParams.get('token')?.trim()
		return token || 'paste-generated-token-here'
	})

	const installCommand = $derived.by(
		() =>
			`curl -fsSL https://raw.githubusercontent.com/hildanku/xemarify/main/deploy/agent/install-agent.sh -o install-agent.sh
chmod +x install-agent.sh
sudo MANAGER_ENDPOINT=${managerEndpoint} ENROLLMENT_TOKEN=${enrollmentToken} ./install-agent.sh`,
	)

	const sampleConfig = $derived.by(
		() =>
			`server:
  endpoint: "${managerEndpoint}"
  insecure: false

enrollment_token: "${enrollmentToken}"

disk_buffer:
  path: "/var/lib/xemarify-agent/spool/events.log"
  max_bytes: 524288000

agent:
  id: ""
  agent_secret: ""
  name: "web-01"
  hostname: "web-01"
  ip_address: "10.0.0.12"

syslog:
  listen: ":5514"

filelog:
  enabled: true
  poll_interval: 5s
  paths:
    - /var/log/syslog
    - /var/log/auth.log

inventory:
  enabled: true
  interval: 5m`,
	)

	const serviceUnit = `[Unit]
Description=Xemarify Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/xemarify-agent
Restart=always
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/etc/xemarify-agent /var/lib/xemarify-agent
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target`

	const targetPaths = [
		{ label: 'Binary', path: '/usr/local/bin/xemarify-agent' },
		{ label: 'Config', path: '/etc/xemarify-agent/agent.yaml' },
		{ label: 'Service', path: '/etc/systemd/system/xemarify-agent.service' },
		{ label: 'Spool', path: '/var/lib/xemarify-agent/spool/events.log' },
	]

	const installSteps = [
		'Generate an enrollment token from the Enrollment Tokens page.',
		'Connect to the target VPS with a privileged shell.',
		'Run the install command above.',
		'Open <code>/etc/xemarify-agent/agent.yaml</code> and adjust hostname, IP, and log sources if needed.',
		'Check service status: <code>systemctl status xemarify-agent</code>',
		'Confirm the new host appears on the Agents page and begins sending heartbeat/events.',
	]

	const firstRunSteps = [
		'Agent starts with <code>enrollment_token</code> present.',
		'Agent calls <code>POST /api/v1/agents/register</code> using <code>X-Enrollment-Token</code>.',
		'Manager returns <code>agent_id</code> and <code>agent_secret</code>.',
		'Agent writes those values back into <code>/etc/xemarify-agent/agent.yaml</code>.',
		'Agent clears <code>enrollment_token</code> and continues runtime auth with <code>X-Agent-Secret</code>.',
	]
</script>

<div class="flex flex-1 flex-col gap-6 p-6 max-w-5xl">

	<!-- Header -->
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Agent Onboarding</h1>
			<p class="text-sm text-muted-foreground mt-1">
				Install and auto-enroll a new Xemarify agent on a VPS.
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" href="/management/enrollment-tokens">
				Generate Token
			</Button>
			<Button variant="outline" size="sm" href="/management/agents">
				View Agents
			</Button>
		</div>
	</div>

	<!-- Install Command -->
	<Card.Root>
		<Card.Header class="pb-3">
			<Card.Title class="text-base">Install Command</Card.Title>
			<Card.Description>
				Run this on the target VPS. Append <code class="text-xs bg-muted px-1 py-0.5 rounded">--insecure</code> only if the manager uses self-signed TLS.
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<pre class="rounded-md border bg-muted p-3 text-xs overflow-x-auto leading-relaxed"><code>{installCommand}</code></pre>
		</Card.Content>
	</Card.Root>

	<!-- Steps + Paths -->
	<div class="grid gap-4 md:grid-cols-2">
		<Card.Root>
			<Card.Header class="pb-3">
				<Card.Title class="text-base">Install Steps</Card.Title>
			</Card.Header>
			<Card.Content>
				<ol class="space-y-2.5 text-sm list-none pl-0">
					{#each installSteps as step, i}
						<li class="flex gap-3">
							<span class="flex-shrink-0 w-5 h-5 rounded-full bg-muted text-muted-foreground text-xs flex items-center justify-center font-medium mt-0.5">{i + 1}</span>
							<span class="text-muted-foreground leading-relaxed">{@html step}</span>
						</li>
					{/each}
				</ol>
			</Card.Content>
		</Card.Root>

		<div class="flex flex-col gap-4">
			<Card.Root>
				<Card.Header class="pb-3">
					<Card.Title class="text-base">Target Paths</Card.Title>
				</Card.Header>
				<Card.Content class="space-y-1">
					{#each targetPaths as { label, path }}
						<div class="flex items-center justify-between py-1.5 border-b last:border-0 gap-4">
							<span class="text-xs text-muted-foreground shrink-0">{label}</span>
							<code class="text-xs text-right truncate">{path}</code>
						</div>
					{/each}
				</Card.Content>
			</Card.Root>

			<Card.Root>
				<Card.Header class="pb-3">
					<Card.Title class="text-base">First-Run Behavior</Card.Title>
				</Card.Header>
				<Card.Content>
					<ol class="space-y-2 text-sm list-none pl-0">
						{#each firstRunSteps as step, i}
							<li class="flex gap-3">
								<span class="flex-shrink-0 w-5 h-5 rounded-full bg-muted text-muted-foreground text-xs flex items-center justify-center font-medium mt-0.5">{i + 1}</span>
								<span class="text-muted-foreground leading-relaxed text-xs">{@html step}</span>
							</li>
						{/each}
					</ol>
				</Card.Content>
			</Card.Root>
		</div>
	</div>

	<!-- Config + Service Unit -->
	<div class="grid gap-4 md:grid-cols-2">
		<Card.Root>
			<Card.Header class="pb-3">
				<Card.Title class="text-base">Sample Config</Card.Title>
				<Card.Description class="text-xs">
					<code>/etc/xemarify-agent/agent.yaml</code>
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<pre class="rounded-md border bg-muted p-3 text-xs overflow-x-auto leading-relaxed"><code>{sampleConfig}</code></pre>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header class="pb-3">
				<Card.Title class="text-base">Systemd Service Unit</Card.Title>
				<Card.Description class="text-xs">
					<code>/etc/systemd/system/xemarify-agent.service</code>
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<pre class="rounded-md border bg-muted p-3 text-xs overflow-x-auto leading-relaxed"><code>{serviceUnit}</code></pre>
			</Card.Content>
		</Card.Root>
	</div>

</div>