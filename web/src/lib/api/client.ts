/**
 * SemarSIEM API Client
 * REST API client untuk komunikasi dengan backend Go
 */

import {
	API_BASE_URL,
	type Event,
	type Alert,
	type Rule,
	type Agent,
	type ListResponse,
	type SingleResponse,
	type ErrorResponse,
	type HealthResponse,
	type PaginationParams,
	type AlertsQueryParams,
	type UpdateAlertStatusRequest,
	type APIResponse,
	type PaginatedResults
} from '$lib/types/api';

class ApiClient {
	private baseURL: string;

	constructor(baseURL: string = API_BASE_URL) {
		this.baseURL = baseURL;
	}

	private async request<T>(
		endpoint: string,
		options?: RequestInit
	): Promise<T> {
		const url = `${this.baseURL}${endpoint}`;
		
		try {
			const response = await fetch(url, {
				...options,
				headers: {
					'Content-Type': 'application/json',
					...options?.headers
				}
			});

			if (!response.ok) {
				const error: ErrorResponse = await response.json();
				throw new Error(error.error || `HTTP ${response.status}`);
			}

			return await response.json();
		} catch (error) {
			if (error instanceof Error) {
				throw error;
			}
			throw new Error('Unknown error occurred');
		}
	}

	// ========== Health Check ==========
	async getHealth(): Promise<HealthResponse> {
		return this.request<HealthResponse>('/health');
	}

	// ========== Events API ==========
	async getEvents(params?: PaginationParams): Promise<ListResponse<Event>> {
		const query = new URLSearchParams();
		if (params?.limit) query.append('limit', params.limit.toString());
		if (params?.offset) query.append('offset', params.offset.toString());
		
		const endpoint = `/api/v1/events${query.toString() ? `?${query}` : ''}`;
		const response = await this.request<any>(endpoint);
		
		// Unwrap APIResponse structure
		if (response.results) {
			return response.results as ListResponse<Event>;
		}
		return response as ListResponse<Event>;
	}

	async getEventById(id: number): Promise<SingleResponse<Event>> {
		const response = await this.request<any>(`/api/v1/events/${id}`);
		
		// Unwrap APIResponse structure
		if (response.results) {
			return { data: response.results } as SingleResponse<Event>;
		}
		return response as SingleResponse<Event>;
	}

	// ========== Alerts API ==========
	async getAlerts(params?: AlertsQueryParams): Promise<ListResponse<Alert>> {
		const query = new URLSearchParams();
		if (params?.limit) query.append('limit', params.limit.toString());
		if (params?.offset) query.append('offset', params.offset.toString());
		if (params?.status) query.append('status', params.status);
		
		const endpoint = `/api/v1/alerts${query.toString() ? `?${query}` : ''}`;
		return this.request<ListResponse<Alert>>(endpoint);
	}

	async getAlertById(id: number): Promise<SingleResponse<Alert>> {
		return this.request<SingleResponse<Alert>>(`/api/v1/alerts/${id}`);
	}

	async updateAlertStatus(
		id: number,
		status: UpdateAlertStatusRequest
	): Promise<{ message: string }> {
		return this.request<{ message: string }>(`/api/v1/alerts/${id}/status`, {
			method: 'PATCH',
			body: JSON.stringify(status)
		});
	}

	// ========== Rules API ==========
	async getRules(params?: PaginationParams): Promise<ListResponse<Rule>> {
		const query = new URLSearchParams();
		if (params?.limit) query.append('limit', params.limit.toString());
		if (params?.offset) query.append('offset', params.offset.toString());
		
		const endpoint = `/api/v1/rules${query.toString() ? `?${query}` : ''}`;
		return this.request<ListResponse<Rule>>(endpoint);
	}

	async getRuleById(id: string): Promise<SingleResponse<Rule>> {
		return this.request<SingleResponse<Rule>>(`/api/v1/rules/${id}`);
	}

	// ========== Agents API ==========
	async getAgents(params?: PaginationParams): Promise<ListResponse<Agent>> {
		const query = new URLSearchParams();
		if (params?.limit) query.append('limit', params.limit.toString());
		if (params?.offset) query.append('offset', params.offset.toString());
		
		const endpoint = `/api/v1/agents${query.toString() ? `?${query}` : ''}`;
		return this.request<ListResponse<Agent>>(endpoint);
	}

	async getAgentById(id: string): Promise<SingleResponse<Agent>> {
		return this.request<SingleResponse<Agent>>(`/api/v1/agents/${id}`);
	}
}

// Export singleton instance
export const apiClient = new ApiClient();

// Export class untuk custom instances
export { ApiClient };
