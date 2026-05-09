import { resolve } from '$app/paths'
import type { SessionUser } from '$lib/auth/session'

const AUTHENTICATED_ONLY_PREFIXES = ['/management', '/access-limited'] as const

const MANAGER_ONLY_PREFIXES = [
	'/management/users',
	'/management/agents',
	'/management/enrollment-tokens',
	'/management/agent-onboarding',
	'/management/rules',
] as const

export function canAccessPath(pathname: string, user: SessionUser | null) {
	const requiresAuth = AUTHENTICATED_ONLY_PREFIXES.some(
		(prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
	)
	if (!requiresAuth) return true
	if (!user) return false

	if (pathname === '/access-limited') return true

	if (MANAGER_ONLY_PREFIXES.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`))) {
		return user.role === 'MANAGER'
	}

	return user.role === 'MANAGER' || user.role === 'ANALYST'
}

export function getDefaultRouteForUser(user: SessionUser | null) {
	if (!user) return resolve('/auth/login')
	if (user.role === 'MANAGER') return resolve('/management')
	if (user.role === 'ANALYST') return resolve('/management/alerts')
	return resolve('/access-limited')
}
