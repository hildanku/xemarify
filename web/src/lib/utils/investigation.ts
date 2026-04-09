export type InvestigationQueryValue = string | undefined | null

export function buildInvestigationHref(
	basePath: string,
	origin: string,
	query: Record<string, InvestigationQueryValue>,
) {
	const target = new URL(basePath, origin)

	for (const [key, value] of Object.entries(query)) {
		if (!value) continue
		target.searchParams.set(key, value)
	}

	return `${target.pathname}${target.search}`
}

export interface TimelineEventLike {
	id: string
	event_time?: string
	received_at?: string
	message: string
	source_ip?: string
	hostname?: string
	category?: string
	severity?: string
}

function toTime(value?: string) {
	if (!value) return 0
	const t = Date.parse(value)
	return Number.isNaN(t) ? 0 : t
}

export function toEventTimeline<T extends TimelineEventLike>(events: T[]) {
	return [...events].sort((a, b) => {
		const left = toTime(a.event_time || a.received_at)
		const right = toTime(b.event_time || b.received_at)
		return right - left
	})
}
