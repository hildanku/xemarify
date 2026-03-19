export const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8089'

export const ACCESS_TOKEN_KEY = 'access_token'
export const REFRESH_TOKEN_KEY = 'refresh_token'

export const V1_BASE_URL = `${BASE_URL}/api/v1`

export const SEARCH_DEBOUNCE_MS = 400

export const LIMIT_OPTIONS = [10, 25, 50, 100] as const

export interface TableParams {
    page: number
    limit: number
    sort: string
    order: 'asc' | 'desc'
    search: string
}

export const DEFAULT_TABLE_PARAMS: TableParams = {
    page: 1,
    limit: 10,
    sort: 'created_at',
    order: 'desc',
    search: '',
}
