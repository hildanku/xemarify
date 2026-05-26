import { browser } from '$app/environment'
import { V1_BASE_URL } from '$lib/constant'
import { readAccessToken } from '$lib/auth/token'

export type SSEEventHandler = (event: MessageEvent) => void
export type SSEStatusHandler = (status: 'connected' | 'connecting' | 'disconnected') => void

export interface SSEClientOptions {
	/** Called when a named event is received. */
	onEvent?: Record<string, SSEEventHandler>
	/** Called when connection status changes. */
	onStatus?: SSEStatusHandler
	/** Reconnect delay in ms after an error (default: 3000). */
	reconnectDelay?: number
}

/**
 * Creates an SSE connection to the events stream endpoint.
 * Uses the access token from localStorage as a query parameter since
 * the EventSource API does not support custom headers.
 *
 * Returns a cleanup function to close the connection.
 */
export function createEventSource(options: SSEClientOptions = {}): () => void {
	if (!browser) return () => {}

	const { onEvent = {}, onStatus, reconnectDelay = 3000 } = options

	let eventSource: EventSource | null = null
	let reconnectTimer: ReturnType<typeof setTimeout> | null = null
	let closed = false

	function connect() {
		if (closed) return

		const token = readAccessToken()
		if (!token) {
			onStatus?.('disconnected')
			// Retry after delay in case user logs in.
			reconnectTimer = setTimeout(connect, reconnectDelay)
			return
		}

		onStatus?.('connecting')

		const url = `${V1_BASE_URL}/events/stream?token=${encodeURIComponent(token)}`
		eventSource = new EventSource(url)

		eventSource.onopen = () => {
			onStatus?.('connected')
		}

		// Register named event listeners.
		for (const [eventType, handler] of Object.entries(onEvent)) {
			eventSource.addEventListener(eventType, handler as EventListener)
		}

		// Also listen for generic messages (unnamed events) as fallback.
		eventSource.onmessage = (event: MessageEvent) => {
			// If the server sends without an event type, try to parse and dispatch.
			if (onEvent['message']) {
				onEvent['message'](event)
			}
		}

		eventSource.onerror = () => {
			eventSource?.close()
			eventSource = null
			onStatus?.('disconnected')

			if (!closed) {
				reconnectTimer = setTimeout(connect, reconnectDelay)
			}
		}
	}

	connect()

	// Return cleanup function.
	return () => {
		closed = true
		if (reconnectTimer) {
			clearTimeout(reconnectTimer)
			reconnectTimer = null
		}
		if (eventSource) {
			eventSource.close()
			eventSource = null
		}
		onStatus?.('disconnected')
	}
}
