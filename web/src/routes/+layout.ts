import { resolve } from '$app/paths'
import { browser } from '$app/environment'
import { redirect } from '@sveltejs/kit'
import { bootstrapSession } from '$lib/auth/session'
import { canAccessPath, getDefaultRouteForUser } from '$lib/auth/guard'
import { bootstrapSystemState } from '$lib/setup/system'

export const load = async ({ url }: { url: URL }) => {
	if (!browser) {
		return {}
	}

	const [systemState, state] = await Promise.all([
		bootstrapSystemState(),
		bootstrapSession(),
	])
	const isSetupPage = url.pathname === resolve('/setup')
	const isLoginPage = url.pathname === resolve('/auth/login')

	if (!systemState.initialized) {
		if (!isSetupPage) {
			throw redirect(302, resolve('/setup'))
		}

		return {}
	}

	if (isSetupPage) {
		throw redirect(
			302,
			state.status === 'authenticated'
				? getDefaultRouteForUser(state.user)
				: resolve('/auth/login'),
		)
	}

	if (state.status === 'authenticated' && isLoginPage) {
		throw redirect(302, getDefaultRouteForUser(state.user))
	}

	if (
		state.status === 'authenticated' &&
		url.pathname === resolve('/access-limited') &&
		state.user?.role !== 'VIEWER'
	) {
		throw redirect(302, getDefaultRouteForUser(state.user))
	}

	if (!canAccessPath(url.pathname, state.user)) {
		if (state.status !== 'authenticated') {
			const loginUrl = new URL(resolve('/auth/login'), url.origin)
			loginUrl.searchParams.set('redirect', `${url.pathname}${url.search}`)
			throw redirect(302, `${loginUrl.pathname}${loginUrl.search}`)
		}

		throw redirect(302, getDefaultRouteForUser(state.user))
	}

	return {}
}
