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