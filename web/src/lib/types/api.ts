/**
 * SemarSIEM API Types
 * Generated from BACKEND_API_CONTEXT.md
 */

// Base API URL
export const API_BASE_URL = 'http://localhost:9000';

// ========== Event Types ==========
export interface Event {
	id: number;
	event_time: string | null;
	received_at: string;
	agent_id: string | null;
	hostname: string | null;
	source_ip: string | null;
	input_type: string | null;
	facility: string | null;
	severity: string | null;
	category: string | null;
	action: string | null;
	status: string | null;
	rule_id: string | null;
	level: number | null;
	message: string;
	normalized: {
		source?: {
			hostname?: string;
			ip?: string;
			input_type?: string;
		};
		classification?: {
			category?: string;
			severity?: string;
			action?: string;
			status?: string;
		};
		actor?: {
			username?: string;
			src_ip?: string;
			src_port?: number;
		};
	};
	raw: string | null;
	created_at: string;
}

// ========== Alert Types ==========
export type AlertStatus = 'new' | 'acknowledged' | 'investigating' | 'resolved' | 'false_positive';

export interface Alert {
	id: number;
	event_id: number;
	rule_id: string;
	level: number;
	status: AlertStatus;
	note: string | null;
	created_at: string;
}

export interface UpdateAlertStatusRequest {
	status: AlertStatus;
}

// ========== Rule Types ==========
export interface Rule {
	id: string;
	name: string;
	description: string | null;
	level: number;
	enabled: boolean;
	condition: Record<string, unknown>;
	tags: string[];
	created_at: string;
	updated_at: string;
}

// ========== Agent Types ==========
export interface Agent {
	id: string;
	name: string;
	hostname: string | null;
	ip_address: string | null;
	version: string | null;
	created_at: string;
	last_seen_at: string | null;
}

// ========== API Response Types ==========
export interface PaginatedResults<T> {
	data: T[];
	limit: number;
	offset: number;
	count: number;
}

export interface APIResponse<T> {
	message: string;
	results?: T;
	error?: {
		code: string;
		message: string;
		details?: Record<string, string>;
		trace?: string;
	};
}

export interface ListResponse<T> {
	data: T[];
	limit: number;
	offset: number;
	count: number;
}

export interface SingleResponse<T> {
	data: T;
}

export interface ErrorResponse {
	error: string;
}

export interface HealthResponse {
	status: string;
	message: string;
	open_connections: string;
	in_use: string;
	idle: string;
}

// ========== Query Parameters ==========
export interface PaginationParams {
	limit?: number;
	offset?: number;
}

export interface AlertsQueryParams extends PaginationParams {
	status?: AlertStatus;
}

// ========== Severity Levels ==========
export const SEVERITY_LEVELS = {
	LOW: { min: 0, max: 3, label: 'Low', color: 'blue' },
	MEDIUM: { min: 4, max: 7, label: 'Medium', color: 'yellow' },
	HIGH: { min: 8, max: 11, label: 'High', color: 'orange' },
	CRITICAL: { min: 12, max: 15, label: 'Critical', color: 'red' }
} as const;

export function getSeverityLevel(level: number) {
	if (level <= 3) return SEVERITY_LEVELS.LOW;
	if (level <= 7) return SEVERITY_LEVELS.MEDIUM;
	if (level <= 11) return SEVERITY_LEVELS.HIGH;
	return SEVERITY_LEVELS.CRITICAL;
}

// ========== Alert Status Colors ==========
export const ALERT_STATUS_COLORS: Record<AlertStatus, string> = {
	new: 'red',
	acknowledged: 'yellow',
	investigating: 'blue',
	resolved: 'green',
	false_positive: 'gray'
};
