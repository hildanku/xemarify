/**
 * Formats a megabyte value into a human-readable string.
 * e.g. 512 → "512 MB", 2048 → "2.0 GB"
 */
export function formatBytes(mb: number): string {
	if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`
	return `${mb} MB`
}

/**
 * Formats a duration in seconds into a human-readable string.
 * e.g. 3661 → "1h 1m", 90061 → "1d 1h 1m"
 */
export function formatUptime(seconds: number): string {
	const d = Math.floor(seconds / 86400)
	const h = Math.floor((seconds % 86400) / 3600)
	const m = Math.floor((seconds % 3600) / 60)
	if (d > 0) return `${d}d ${h}h ${m}m`
	if (h > 0) return `${h}h ${m}m`
	return `${m}m`
}
