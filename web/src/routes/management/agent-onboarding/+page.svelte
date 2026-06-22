<script lang="ts">
	import { page } from '$app/state'
	import { Button } from '$lib/components/ui/button/index.js'
	import * as Card from '$lib/components/ui/card/index.js'
	import { Badge } from '$lib/components/ui/badge/index.js'

	const managerEndpoint = $derived.by(() => {
		const endpoint = page.url.searchParams.get('endpoint')?.trim()
		return endpoint || 'https://manager.example.com'
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

	// These are hardcoded strings, not user input — no XSS risk
	const installSteps = [
		'Generate an enrollment token from the Enrollment Tokens page.',
		'Connect to the target VPS with a privileged shell.',
		'Run the install command above.',
		'Open <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">/etc/xemarify-agent/agent.yaml</code> and adjust hostname, IP, and log sources if needed.',
		'Check service status: <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">systemctl status xemarify-agent</code>',
		'Confirm the new host appears on the Agents page and begins sending heartbeat/events.',
	]

	const firstRunSteps = [
		'Agent starts with <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">enrollment_token</code> present.',
		'Agent calls <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">POST /api/v1/agents/register</code> using <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">X-Enrollment-Token</code>.',
		'Manager returns <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">agent_id</code> and <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">agent_secret</code>.',
		'Agent writes those values back into <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">/etc/xemarify-agent/agent.yaml</code>.',
		'Agent clears <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">enrollment_token</code> and continues runtime auth with <code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">X-Agent-Secret</code>.',
	]
</script>

<div class="flex flex-1 flex-col gap-6 p-4 md:p-6 max-w-full">

	<!-- Header -->
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Agent Onboarding</h1>
			<p class="text-muted-foreground mt-1">
				Install and auto-enroll a new Xemarify agent on a VPS.
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
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
		<Card.Header>
			<Card.Title>Install Command</Card.Title>
			<Card.Description>
				Run this on the target VPS. Append
				<code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">--insecure</code>
				only if the manager uses self-signed TLS.
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<pre class="rounded-md border bg-muted p-4 text-xs overflow-x-auto leading-relaxed whitespace-pre"><code>{installCommand}</code></pre>
		</Card.Content>
	</Card.Root>

	<!-- Steps + Paths -->
	<div class="grid gap-4 lg:grid-cols-2">
		<Card.Root>
			<Card.Header>
				<Card.Title>Install Steps</Card.Title>
				<Card.Description>Follow these steps to get the agent running.</Card.Description>
			</Card.Header>
			<Card.Content>
				<ol class="space-y-3 text-sm list-none pl-0">
					{#each installSteps as step, i (i)}
						<li class="flex gap-3">
							<span class="flex-shrink-0 w-6 h-6 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center font-semibold mt-0.5">{i + 1}</span>
							<span class="text-muted-foreground leading-relaxed">{@html step}</span>
						</li>
					{/each}
				</ol>
			</Card.Content>
		</Card.Root>

		<div class="flex flex-col gap-4">
			<Card.Root>
				<Card.Header>
					<Card.Title>Target Paths</Card.Title>
					<Card.Description>Files written during installation.</Card.Description>
				</Card.Header>
				<Card.Content class="pt-0">
					{#each targetPaths as { label, path } (label)}
						<div class="flex items-center justify-between py-2.5 border-b last:border-0 gap-4">
							<span class="text-sm text-muted-foreground shrink-0 w-16">{label}</span>
							<code class="font-mono text-xs text-right truncate">{path}</code>
						</div>
					{/each}
				</Card.Content>
			</Card.Root>

			<Card.Root>
				<Card.Header>
					<Card.Title>First-Run Behavior</Card.Title>
					<Card.Description>How the agent self-registers on first boot.</Card.Description>
				</Card.Header>
				<Card.Content>
					<ol class="space-y-3 text-sm list-none pl-0">
						{#each firstRunSteps as step, i (i)}
							<li class="flex gap-3">
								<span class="flex-shrink-0 w-6 h-6 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center font-semibold mt-0.5">{i + 1}</span>
								<span class="text-muted-foreground leading-relaxed text-xs">{@html step}</span>
							</li>
						{/each}
					</ol>
				</Card.Content>
			</Card.Root>
		</div>
	</div>

	<!-- Config + Service Unit -->
	<div class="grid gap-4 lg:grid-cols-2">
		<Card.Root>
			<Card.Header>
				<Card.Title>Sample Config</Card.Title>
				<Card.Description>
					<code class="font-mono text-xs">/etc/xemarify-agent/agent.yaml</code>
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<pre class="rounded-md border bg-muted p-4 text-xs overflow-x-auto leading-relaxed whitespace-pre"><code>{sampleConfig}</code></pre>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Systemd Service Unit</Card.Title>
				<Card.Description>
					<code class="font-mono text-xs">/etc/systemd/system/xemarify-agent.service</code>
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<pre class="rounded-md border bg-muted p-4 text-xs overflow-x-auto leading-relaxed whitespace-pre"><code>{serviceUnit}</code></pre>
			</Card.Content>
		</Card.Root>
	</div>

</div>

<svelte:head>
	<title>Xemarify - Agent Onboarding</title>
</svelte:head>
