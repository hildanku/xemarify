export type AgentStatus = 'ONLINE' | 'OFFLINE'

export interface Agent {
	id: string
	name: string
	hostname: string | null
	ip_address: string | null
	version: string | null
	status: AgentStatus
	created_at: string
	last_seen_at: string | null
}

export interface CreateAgentRequest {
	name: string
	hostname?: string
	ip_address?: string
	version?: string
	status?: AgentStatus
	agent_secret?: string
}

export interface UpdateAgentRequest {
	name: string
	hostname?: string
	ip_address?: string
	version?: string
	status: AgentStatus
}

export type UserRole = 'MANAGER' | 'ANALYST' | 'VIEWER'

export interface User {
	id: string
	username: string
	email: string
	role: UserRole
	avatar?: string | null
	created_at: string
	updated_at?: string | null
}

export interface CreateUserRequest {
	username: string
	email: string
	role: UserRole
	password: string
	avatar?: string
}

export interface UpdateUserRequest {
	username?: string
	email?: string
	role?: UserRole
	avatar?: string | null
}

export type RuleSeverity = 'INFO' | 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'

export type AlertStatus = 'new' | 'acknowledged' | 'closed'

export interface RuleCondition {
	type?: 'threshold' | 'sequence' | 'correlation' | 'anomaly'
	event_type?: string
	group_by: string[]
	threshold?: number
	window_sec?: number
	severity?: RuleSeverity
	sequence_steps?: string[]
	correlation_event_types?: string[]
	min_distinct_event_types?: number
	baseline_window_sec?: number
	spike_factor?: number
	anomaly_min_count?: number
}

export interface Rule {
	id: string
	name: string
	description?: string
	level: RuleSeverity
	enabled: boolean
	condition: RuleCondition
	tags: string[]
	version: number
	created_by?: string
	created_at: string
	updated_at: string
}

export interface CreateRuleRequest {
	name: string
	description?: string
	level: RuleSeverity
	enabled: boolean
	condition: RuleCondition
	tags?: string[]
}

export interface UpdateRuleRequest {
	name?: string
	description?: string
	level?: RuleSeverity
	enabled?: boolean
	condition?: RuleCondition
	tags?: string[]
}

export interface Alert {
	id: string
	rule_id: string
	rule_name: string
	severity: RuleSeverity
	correlation_key: string
	triggered_at: string
	status: AlertStatus
	created_at: string
}

export interface AlertEvent {
	id: string
	event_time: string
	received_at: string
	agent_id: string
	hostname?: string
	source_ip?: string
	input_type?: string
	facility?: string
	severity?: string
	category?: string
	message: string
	normalized?: Record<string, unknown>
	raw?: string
}

export interface AlertDetail {
	alert: Alert
	events: AlertEvent[]
}

export interface EventItem {
	id: string
	event_time: string
	received_at: string
	agent_id: string
	hostname: string
	source_ip?: string
	input_type?: string
	facility?: string
	severity?: string
	category?: string
	message: string
	normalized?: Record<string, unknown>
}

export interface EventDetail extends EventItem {
	raw?: string
}

export interface AuditLog {
	id: string
	user_id?: string | null
	user_identifier: string
	action: string
	object_type?: string | null
	object_id?: string | null
	metadata?: Record<string, unknown>
	created_at: string
}
