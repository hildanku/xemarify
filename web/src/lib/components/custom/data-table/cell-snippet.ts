import { createRawSnippet } from 'svelte'

export const cellSnippet = createRawSnippet<[{ value: string; class?: string }]>(
	(getProps) => ({
		render: () =>
			`<span class="text-sm ${getProps().class ?? ''}">${getProps().value}</span>`,
	}),
)
