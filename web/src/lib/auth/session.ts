import { browser } from '$app/environment'
import { get, writable } from 'svelte/store'
import { BASE_URL } from '$lib/constant'
import type { ApiResponse } from '$lib/client'
import type { UserRole } from '$lib/types/api'
import {
	clearTokens,
	decodeJwt,
	isTokenExpired,
	readTokens,
	setTokens,
} from '$lib/auth/token'

export interface SessionUser {
	userId: string
	username: string
	role: UserRole
	email?: string
}

export interface AuthState {
	initialized: boolean
	status: 'loading' | 'authenticated' | 'anonymous'
	user: SessionUser | null
}

interface AuthTokensResponse {
	access_token: string
	refresh_token: string
}

export const auth = writable<AuthState>({
	initialized: false,
	status: 'loading',
	user: null,
})

let bootstrapPromise: Promise<AuthState> | null = null
let refreshPromise: Promise<string | null> | null = null

export function writeSession(accessToken: string, refreshToken: string) {
	setTokens(accessToken, refreshToken)
	applySessionFromToken(accessToken)
}

function clearStoredSession() {
	clearTokens()
}

function buildUserFromToken(token: string): SessionUser | null {
	const payload = decodeJwt(token)
	if (!payload?.user_id || !payload.username || !payload.role) {
		return null
	}

	return {
		userId: payload.user_id,
		username: payload.username,
		role: payload.role,
	}
}

export function applySessionFromToken(accessToken: string) {
	const user = buildUserFromToken(accessToken)
	if (!user) {
		clearSession()
		return
	}

	auth.set({
		initialized: true,
		status: 'authenticated',
		user,
	})
}

export function clearSession() {
	clearStoredSession()
	auth.set({
		initialized: true,
		status: 'anonymous',
		user: null,
	})
}

async function parseApiResponse<T>(
	response: Response,
): Promise<ApiResponse<T>> {
	const text = await response.text()
	const data = text
		? (JSON.parse(text) as ApiResponse<T>)
		: ({ message: '', data: null } as ApiResponse<T>)

	if (!response.ok) {
		throw new Error(data.message || `HTTP error! status: ${response.status}`)
	}

	return data
}

export async function refreshSession() {
	if (!browser) return null
	if (refreshPromise) return refreshPromise

	refreshPromise = (async () => {
		const { refreshToken } = readTokens()
		if (!refreshToken) {
			clearSession()
			return null
		}

		try {
			const response = await fetch(`${BASE_URL}/auth/refresh`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ refresh_token: refreshToken }),
			})

			const result = await parseApiResponse<AuthTokensResponse>(response)
			writeSession(result.data.access_token, result.data.refresh_token)
			return result.data.access_token
		} catch {
			clearSession()
			return null
		} finally {
			refreshPromise = null
		}
	})()

	return refreshPromise
}

export async function bootstrapSession(force = false) {
	if (!browser) {
		return get(auth)
	}

	const state = get(auth)
	if (!force && state.initialized && state.status !== 'loading') {
		return state
	}

	if (bootstrapPromise && !force) return bootstrapPromise

	auth.update((current) => ({
		...current,
		status: 'loading',
	}))

	bootstrapPromise = (async () => {
		const { accessToken, refreshToken } = readTokens()
		if (accessToken && !isTokenExpired(accessToken)) {
			applySessionFromToken(accessToken)
			return get(auth)
		}

		if (refreshToken) {
			await refreshSession()
			return get(auth)
		}

		clearSession()
		return get(auth)
	})()

	try {
		return await bootstrapPromise
	} finally {
		bootstrapPromise = null
	}
}

export async function login(email: string, password: string) {
	const response = await fetch(`${BASE_URL}/auth/login`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({ email, password }),
	})

	const result = await parseApiResponse<AuthTokensResponse>(response)
	writeSession(result.data.access_token, result.data.refresh_token)
	return get(auth)
}

export async function logout(options?: { remote?: boolean }) {
	const remote = options?.remote ?? true

	try {
		if (remote) {
			const { accessToken, refreshToken } = readTokens()
			const token =
				accessToken && !isTokenExpired(accessToken)
					? accessToken
					: refreshToken
						? await refreshSession()
						: null
			if (token) {
				await fetch(`${BASE_URL}/auth/logout`, {
					method: 'POST',
					headers: {
						Authorization: `Bearer ${token}`,
					},
				})
			}
		}
	} catch {
	} finally {
		clearSession()
	}
}
