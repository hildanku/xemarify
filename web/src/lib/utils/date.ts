const LOCALE = 'id-ID'

/**
 * "21 Jun 2026, 21.11"
 * For ISO string, Date object, or epoch ms number
 */
export function formatDateTime(value: string | Date | number | null | undefined): string {
	if (!value) return '—'
	return new Date(value).toLocaleString(LOCALE, {
		dateStyle: 'medium',
		timeStyle: 'short',
	})
}

/**
 * "21 Jun 2026"
 */
export function formatDate(value: string | Date | number | null | undefined): string {
	if (!value) return '—'
	return new Date(value).toLocaleDateString(LOCALE, {
		day: '2-digit',
		month: 'short',
		year: 'numeric',
	})
}

/**
 * "21.11"
 */
export function formatTime(value: string | Date | number | null | undefined): string {
	if (!value) return '—'
	return new Date(value).toLocaleTimeString(LOCALE, {
		hour: '2-digit',
		minute: '2-digit',
	})
}

/**
 * "Min, 21"
 * For day labels in charts, where space is limited and month/year is not needed
 */
export function formatDayLabel(isoDate: string): string {
	return new Date(isoDate).toLocaleDateString(LOCALE, {
		weekday: 'short',
		day: '2-digit',
	})
}

/**
 * "21.11.45"
 * For last updated timestamps, where only time is relevant. Input is epoch ms number
 * Returns fallback string if falsy 
 */
export function formatLastUpdated(
	timestamp: number,
	fallback = 'Waiting for telemetry',
): string {
	if (!timestamp) return fallback
	return new Date(timestamp).toLocaleTimeString(LOCALE, {
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
	})
}

/**
 * "21 Jun 2026, 21.11.45" 
 * When exact date and time is needed, such as event details or logs
 */
export function formatDateTimeExact(value: string | Date | number | null | undefined): string {
	if (!value) return '—'
	return new Date(value).toLocaleString(LOCALE, {
		dateStyle: 'medium',
		timeStyle: 'medium',
	})
}

/**
 * Relative time: "5m ago", "2h ago", dll
 */
export function formatRelative(value: string | null | undefined): string {
	if (!value) return ''
	const secs = Math.floor((Date.now() - new Date(value).getTime()) / 1000)
	if (secs < 60) return `${secs}s ago`
	if (secs < 3600) return `${Math.floor(secs / 60)}m ago`
	if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`
	return `${Math.floor(secs / 86400)}d ago`
}
