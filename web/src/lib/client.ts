import { refreshSession } from '$lib/auth/session'
import { isTokenExpired, readTokens } from '$lib/auth/token'

export interface ApiResponse<T> {
	message: string
	data: T
}

export interface ApiResponseWithMetadata<T> {
	message: string
	data: {
		items: T
		metadata: {
			total: number
			total_pages: number
			limit: number
			offset: number
		}
	}
}

/** Cursor-based (keyset) pagination response, this interface used by the events endpoint. */
export interface ApiResponseWithCursorMetadata<T> {
	message: string
	data: {
		items: T
		metadata: {
			next_cursor: string
			has_more: boolean
			limit: number
		}
	}
}

async function parseError(response: Response) {
    const text = await response.text()

    if (!text) {
        return `HTTP error! status: ${response.status}`
    }

    try {
        const data = JSON.parse(text) as { message?: string }
        return data.message || text
    } catch {
        return text
    }
}

async function ensureValidAccessToken() {
    const { accessToken, refreshToken } = readTokens()

    if (accessToken && !isTokenExpired(accessToken)) {
        return accessToken
    }

    if (!refreshToken) {
        return null
    }

    return refreshSession()
}

export async function clientFetch<T>(
	url: string,
	options?: RequestInit,
	config?: { auth?: boolean },
): Promise<T> {
	const headers = new Headers(options?.headers)
	const useAuth = config?.auth !== false

    if (useAuth) {
        const token = await ensureValidAccessToken()
        if (token) {
            headers.set('Authorization', `Bearer ${token}`)
        }
    }

    if (!(options?.body instanceof FormData) && !headers.has('Content-Type')) {
        headers.set('Content-Type', 'application/json')
    }

    let response = await fetch(url, {
        ...options,
        headers,
    })

    if (useAuth && response.status === 401) {
        const token = await refreshSession()
        if (token) {
            headers.set('Authorization', `Bearer ${token}`)
            response = await fetch(url, {
                ...options,
                headers,
            })
        }
    }

	if (!response.ok) {
		const errorText = await parseError(response)
		throw new Error(
			`HTTP error! status: ${response.status}, message: ${errorText}`,
		)
	}

    if (response.status === 204) {
        return undefined as T
    }

    return response.json() as Promise<T>
}
