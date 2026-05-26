import { page } from '$app/stores'
import { get } from 'svelte/store'
import { updateTableParams } from '$lib/utils/table-params'

/**
 * Creates common table navigation handlers.
 * Call this in your page component to get reusable sort/page/limit handlers.
 */
export function createTableHandlers() {
	function handleSortChange(sort: string, order: 'asc' | 'desc') {
		updateTableParams({ sort, order }, get(page).url)
	}

	function gotoPage(p: number) {
		updateTableParams({ page: p }, get(page).url)
	}

	function handleLimitChange(value: string | undefined) {
		if (!value) return
		updateTableParams({ limit: parseInt(value), page: 1 }, get(page).url)
	}

	return { handleSortChange, gotoPage, handleLimitChange }
}
