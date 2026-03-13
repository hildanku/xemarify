import { goto } from '$app/navigation'
import { base } from '$app/paths'
import { DEFAULT_TABLE_PARAMS, type TableParams } from '$lib/constant'

// Re-export for convenience
export type { TableParams } from '$lib/constant'

/**
 * Recalculate which page the user should land on after changing the limit.
 * Keeps the first visible item of the current view in frame where possible.
 */
export function recalculatePage(
    currentPage: number,
    totalItems: number,
    currentOffset: number,
    newLimit: number,
): number {
    if (newLimit <= 0 || totalItems <= 0) return 1
    const firstItemIndex = currentOffset || (currentPage - 1) * newLimit
    return Math.max(1, Math.ceil((firstItemIndex + 1) / newLimit))
}

/**
 * Parse table params from a URL's search params.
 */
export function parseTableParams(url: URL): TableParams {
    return {
        page: Math.max(1, parseInt(url.searchParams.get('page') ?? '1')),
        limit: parseInt(url.searchParams.get('limit') ?? '10'),
        sort: url.searchParams.get('sort') ?? DEFAULT_TABLE_PARAMS.sort,
        order: (url.searchParams.get('order') ?? DEFAULT_TABLE_PARAMS.order) as 'asc' | 'desc',
        search: url.searchParams.get('search') ?? '',
    }
}

/**
 * Merge a partial params update into the current URL and navigate.
 * Uses replaceState so the history stack is not polluted.
 * Automatically resets page to 1 when search, limit, or sort changes.
 */
export function updateTableParams(
    params: Partial<TableParams>,
    currentUrl: URL,
): void {
    const url = new URL(currentUrl.toString())

    const resetPage =
        'search' in params ||
        'limit' in params ||
        ('sort' in params && params.sort !== currentUrl.searchParams.get('sort')) ||
        ('order' in params && params.order !== currentUrl.searchParams.get('order'))

    if (params.page !== undefined) url.searchParams.set('page', String(params.page))
    if (params.limit !== undefined) url.searchParams.set('limit', String(params.limit))
    if (params.sort !== undefined) url.searchParams.set('sort', params.sort)
    if (params.order !== undefined) url.searchParams.set('order', params.order)
    if (params.search !== undefined) {
        if (params.search) {
            url.searchParams.set('search', params.search)
        } else {
            url.searchParams.delete('search')
        }
    }

    if (resetPage && !('page' in params)) {
        url.searchParams.set('page', '1')
    }

    // SvelteKit's static analyzer may warn on dynamic goto paths; safe at runtime.
    goto(`${base}${url.pathname}${url.search}`, { replaceState: true, noScroll: true, keepFocus: true })
}

/**
 * Update arbitrary URL search params and navigate without full page reload.
 */
export function updateSearchParams(
    params: Record<string, string | undefined>,
    currentUrl: URL,
    options?: { resetPage?: boolean },
): void {
    const url = new URL(currentUrl.toString())

    for (const [key, value] of Object.entries(params)) {
        if (value === undefined) continue
        if (value) {
            url.searchParams.set(key, value)
        } else {
            url.searchParams.delete(key)
        }
    }

    if (options?.resetPage) {
        url.searchParams.set('page', '1')
    }

    goto(`${base}${url.pathname}${url.search}`, { replaceState: true, noScroll: true, keepFocus: true })
}

/**
 * Serialize table params into a query string for the API request.
 */
export function buildQueryString(params: TableParams): string {
    const qs = new URLSearchParams()
    qs.set('page', String(params.page))
    qs.set('limit', String(params.limit))
    qs.set('sort', params.sort)
    qs.set('order', params.order)
    if (params.search) qs.set('search', params.search)
    return qs.toString()
}
