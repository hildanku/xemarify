<script lang="ts">
	import { onMount } from 'svelte';
	import { apiClient } from '$lib/api/client';
	import type { Rule } from '$lib/types/api';
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as Table from "$lib/components/ui/table/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import { Switch } from "$lib/components/ui/switch/index.js";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";

	// State
	let rules = $state<Rule[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	
	// Pagination
	let currentPage = $state(1);
	let limit = $state(50);
	let totalCount = $state(0);
	let totalPages = $derived(Math.ceil(totalCount / limit));

	// Selected rule for detail modal
	let selectedRule = $state<Rule | null>(null);
	let dialogOpen = $state(false);

	async function loadRules() {
		try {
			loading = true;
			error = null;
			const offset = (currentPage - 1) * limit;
			const response = await apiClient.getRules({ limit, offset });
			rules = response.data;
			totalCount = response.count;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load rules';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadRules();
	});

	function goToPage(page: number) {
		if (page >= 1 && page <= totalPages) {
			currentPage = page;
			loadRules();
		}
	}

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleString('id-ID');
	}

	function getBadgeVariant(level: number): "default" | "secondary" | "destructive" | "outline" {
		if (level >= 12) return "destructive";
		if (level >= 8) return "default";
		if (level >= 4) return "secondary";
		return "outline";
	}

	function openRuleDetail(rule: Rule) {
		selectedRule = rule;
		dialogOpen = true;
	}

	// TODO: Implement toggle rule enabled/disabled
	function toggleRule(ruleId: string, currentState: boolean) {
		console.log(`Toggle rule ${ruleId} to ${!currentState}`);
		// This would require a PATCH endpoint on the backend
	}
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold">Detection Rules</h1>
			<p class="text-muted-foreground">Manage security detection rules</p>
		</div>
		<Button onclick={() => loadRules()} disabled={loading}>
			<RefreshCwIcon class="h-4 w-4 mr-2" />
			Refresh
		</Button>
	</div>

	{#if error}
		<Card.Root class="border-destructive">
			<Card.Content class="pt-6">
				<p class="text-destructive">{error}</p>
			</Card.Content>
		</Card.Root>
	{/if}

	<!-- Rules Table -->
	<Card.Root>
		<Card.Content class="p-0">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>Name</Table.Head>
						<Table.Head>Description</Table.Head>
						<Table.Head>Level</Table.Head>
						<Table.Head>Tags</Table.Head>
						<Table.Head>Status</Table.Head>
						<Table.Head>Created</Table.Head>
						<Table.Head class="text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#if loading}
						<Table.Row>
							<Table.Cell colspan={7} class="text-center py-8">
								<div class="flex items-center justify-center">
									<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
								</div>
							</Table.Cell>
						</Table.Row>
					{:else if rules.length === 0}
						<Table.Row>
							<Table.Cell colspan={7} class="text-center py-8 text-muted-foreground">
								No rules found
							</Table.Cell>
						</Table.Row>
					{:else}
						{#each rules as rule}
							<Table.Row class="cursor-pointer" onclick={() => openRuleDetail(rule)}>
								<Table.Cell class="font-medium">{rule.name}</Table.Cell>
								<Table.Cell class="max-w-md truncate">{rule.description || 'N/A'}</Table.Cell>
								<Table.Cell>
									<Badge variant={getBadgeVariant(rule.level)}>
										Level {rule.level}
									</Badge>
								</Table.Cell>
								<Table.Cell>
									<div class="flex gap-1 flex-wrap">
										{#each rule.tags.slice(0, 2) as tag}
											<Badge variant="outline" class="text-xs">{tag}</Badge>
										{/each}
										{#if rule.tags.length > 2}
											<Badge variant="outline" class="text-xs">+{rule.tags.length - 2}</Badge>
										{/if}
									</div>
								</Table.Cell>
								<Table.Cell>
									<div class="flex items-center gap-2" onclick={(e) => e.stopPropagation()}>
										<Switch 
											checked={rule.enabled}
											onCheckedChange={() => toggleRule(rule.id, rule.enabled)}
										/>
										<span class="text-sm">{rule.enabled ? 'Enabled' : 'Disabled'}</span>
									</div>
								</Table.Cell>
								<Table.Cell class="text-xs">{formatDate(rule.created_at)}</Table.Cell>
								<Table.Cell class="text-right">
									<Button 
										size="sm"
										variant="ghost"
										onclick={(e) => { e.stopPropagation(); openRuleDetail(rule); }}
									>
										View
									</Button>
								</Table.Cell>
							</Table.Row>
						{/each}
					{/if}
				</Table.Body>
			</Table.Root>
		</Card.Content>

		<!-- Pagination -->
		{#if !loading && rules.length > 0}
			<Card.Footer class="flex items-center justify-between">
				<div class="text-sm text-muted-foreground">
					Showing <span class="font-medium">{(currentPage - 1) * limit + 1}</span> to 
					<span class="font-medium">{Math.min(currentPage * limit, totalCount)}</span> of 
					<span class="font-medium">{totalCount}</span> rules
				</div>
				<div class="flex gap-2">
					<Button
						variant="outline"
						size="sm"
						onclick={() => goToPage(currentPage - 1)}
						disabled={currentPage === 1}
					>
						Previous
					</Button>
					<div class="flex items-center px-3 text-sm">
						Page {currentPage} of {totalPages}
					</div>
					<Button
						variant="outline"
						size="sm"
						onclick={() => goToPage(currentPage + 1)}
						disabled={currentPage === totalPages}
					>
						Next
					</Button>
				</div>
			</Card.Footer>
		{/if}
	</Card.Root>
</div>

<!-- Rule Detail Dialog -->
<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Content class="max-w-3xl max-h-[90vh] overflow-y-auto">
		{#if selectedRule}
			<Dialog.Header>
				<Dialog.Title>{selectedRule.name}</Dialog.Title>
				<Dialog.Description>Rule ID: {selectedRule.id}</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-4">
				<div class="grid grid-cols-2 gap-4">
					<div>
						<p class="text-sm font-medium text-muted-foreground">Rule ID</p>
						<p class="text-base font-mono text-xs">{selectedRule.id}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Level</p>
						<Badge variant={getBadgeVariant(selectedRule.level)}>
							Level {selectedRule.level}
						</Badge>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Status</p>
						<Badge variant={selectedRule.enabled ? "default" : "secondary"}>
							{selectedRule.enabled ? 'Enabled' : 'Disabled'}
						</Badge>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Created</p>
						<p class="text-base text-xs">{formatDate(selectedRule.created_at)}</p>
					</div>
				</div>

				{#if selectedRule.description}
					<div>
						<p class="text-sm font-medium text-muted-foreground mb-2">Description</p>
						<Card.Root>
							<Card.Content class="pt-4">
								<p class="text-base">{selectedRule.description}</p>
							</Card.Content>
						</Card.Root>
					</div>
				{/if}

				<div>
					<p class="text-sm font-medium text-muted-foreground mb-2">Tags</p>
					<div class="flex gap-2 flex-wrap">
						{#each selectedRule.tags as tag}
							<Badge variant="outline">{tag}</Badge>
						{/each}
					</div>
				</div>

				{#if selectedRule.condition && Object.keys(selectedRule.condition).length > 0}
					<div>
						<p class="text-sm font-medium text-muted-foreground mb-2">Condition</p>
						<Card.Root>
							<Card.Content class="pt-4">
								<pre class="text-sm overflow-x-auto">{JSON.stringify(selectedRule.condition, null, 2)}</pre>
							</Card.Content>
						</Card.Root>
					</div>
				{/if}

				<div class="grid grid-cols-2 gap-4">
					<div>
						<p class="text-sm font-medium text-muted-foreground">Created At</p>
						<p class="text-sm">{formatDate(selectedRule.created_at)}</p>
					</div>
					<div>
						<p class="text-sm font-medium text-muted-foreground">Updated At</p>
						<p class="text-sm">{formatDate(selectedRule.updated_at)}</p>
					</div>
				</div>
			</div>
			<Dialog.Footer>
				<Button variant="outline" onclick={() => dialogOpen = false}>Close</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
