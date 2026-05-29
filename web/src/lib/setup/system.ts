import { browser } from '$app/environment'
import { get, writable } from 'svelte/store'
import { BASE_URL } from '$lib/constant'

interface HealthResponse {
	status: string
	initialized: boolean
}

interface SetupTokensResponse {
	access_token: string
	refresh_token: string
}

export interface SystemState {
	checked: boolean
	initialized: boolean
}

export const system = writable<SystemState>({
	checked: false,
	initialized: true,
})

let bootstrapPromise: Promise<SystemState> | null = null

function readErrorMessage(data: unknown) {
	if (!data || typeof data !== 'object') return 'Unknown error'
	const message = Reflect.get(data, 'message')
	return typeof message === 'string' && message ? message : 'Unknown error'
}

export function setSystemInitialized(initialized: boolean) {
	system.set({ checked: true, initialized })
}

export async function bootstrapSystemState(force = false) {
	if (!browser) return get(system)

	const state = get(system)
	if (!force && state.checked) return state
	if (bootstrapPromise && !force) return bootstrapPromise

	bootstrapPromise = (async () => {
		const response = await fetch(`${BASE_URL}/api/health`)
		if (!response.ok) {
			throw new Error(`Failed to load system status (${response.status})`)
		}

		const data = (await response.json()) as HealthResponse
		if (typeof data.initialized !== 'boolean') {
			throw new Error('Invalid system status response')
		}

		setSystemInitialized(data.initialized)
		return get(system)
	})()

	try {
		return await bootstrapPromise
	} finally {
		bootstrapPromise = null
	}
}

export async function initializeFirstManager(input: {
	username: string
	email: string
	password: string
	setupToken: string
}) {
	const response = await fetch(`${BASE_URL}/api/setup/initialize`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({
			username: input.username,
			email: input.email,
			password: input.password,
			setup_token: input.setupToken,
		}),
	})

	const payload = await response.json().catch(() => null)
	if (!response.ok) {
		throw new Error(readErrorMessage(payload))
	}

	setSystemInitialized(true)
	const data = payload as { data: SetupTokensResponse }
	return data.data
}
