export interface ApiResponse<T> {
    message: string
    data: T
}

export interface ApiResponseWithMetadata<T> {
    message: string
    data: {
        items: T,
        metadata: {
            total: number
            total_pages: number
            limit: number
            offset: number
        }
    }
}

export async function clientFetch<T>(url: string, options?: RequestInit): Promise<T> {
    const token = localStorage.getItem('access_token')
    const headers = new Headers(options?.headers)

    if (token) {
        headers.set('Authorization', `Bearer ${token}`)
    }

    const response = await fetch(url, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...headers,
        },
    })

    if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`)
    }
    return response.json() as Promise<T>
}
