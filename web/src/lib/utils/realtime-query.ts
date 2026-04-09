export const REALTIME_INTERVAL_MS = 15000

export function realtimeQueryOptions(enabled = true, intervalMs = REALTIME_INTERVAL_MS) {
	if (!enabled) {
		return {
			refetchOnWindowFocus: true,
		}
	}

	return {
		refetchInterval: intervalMs,
		refetchIntervalInBackground: true,
		refetchOnWindowFocus: true,
	}
}
