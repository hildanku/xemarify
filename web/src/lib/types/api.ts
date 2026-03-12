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

export interface UpsertAgentRequest {
    name: string
    hostname?: string
    ip_address?: string
    version?: string
}
