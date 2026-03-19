import { browser } from '$app/environment'
import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from '$lib/constant'
import type { UserRole } from '$lib/types/api'

export interface SessionTokens {
	accessToken: string | null
	refreshToken: string | null
}

export interface JwtPayload {
	user_id?: string
	username?: string
	role?: UserRole
	exp?: number
}

export function readAccessToken() {
	if (!browser) return null
	return localStorage.getItem(ACCESS_TOKEN_KEY)
}

export function readRefreshToken() {
	if (!browser) return null
	return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function readTokens(): SessionTokens {
	return {
		accessToken: readAccessToken(),
		refreshToken: readRefreshToken(),
	}
}

export function setTokens(accessToken: string, refreshToken: string) {
	if (!browser) return

	localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
	localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
}

export function clearTokens() {
	if (!browser) return

	localStorage.removeItem(ACCESS_TOKEN_KEY)
	localStorage.removeItem(REFRESH_TOKEN_KEY)
}

export function decodeJwt(token: string): JwtPayload | null {
	try {
		const [, payload] = token.split('.')
		if (!payload) return null

		const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
		const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
		return JSON.parse(atob(padded)) as JwtPayload
	} catch {
		return null
	}
}

export function isTokenExpired(token: string, skewSeconds = 30): boolean {
	const payload = decodeJwt(token)
	if (!payload?.exp) return true

	return payload.exp <= Math.floor(Date.now() / 1000) + skewSeconds
}
