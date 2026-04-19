<script lang="ts">
	import { Button } from "$lib/components/ui/button/index.js";
	import MoonIcon from "@lucide/svelte/icons/moon";
	import SunIcon from "@lucide/svelte/icons/sun";
	import { onMount } from "svelte";

	const THEME_STORAGE_KEY = "theme";
	let theme = $state<"light" | "dark">("light");

	const applyTheme = (nextTheme: "light" | "dark") => {
		theme = nextTheme;
		document.documentElement.classList.toggle("dark", nextTheme === "dark");
		document.documentElement.style.colorScheme = nextTheme;
		localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
	};

	const toggleTheme = () => {
		applyTheme(theme === "dark" ? "light" : "dark");
	};

	onMount(() => {
		const storedTheme = localStorage.getItem(THEME_STORAGE_KEY);
		if (storedTheme === "dark" || storedTheme === "light") {
			applyTheme(storedTheme);
			return;
		}

		const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
		applyTheme(prefersDark ? "dark" : "light");
	});
</script>

<Button
	variant="ghost"
	size="icon"
	onclick={toggleTheme}
	aria-label="Toggle dark light mode"
	title="Toggle dark light mode"
>
	{#if theme === "dark"}
		<SunIcon />
	{:else}
		<MoonIcon />
	{/if}
</Button>
