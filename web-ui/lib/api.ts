export const API_BASE = "/api/v1"

// Auth types
export interface AuthResponse {
    token: string
    expires_at: string
    user_id: string
}

export interface MagicLinkResponse {
    message: string
    token: string // Included for dev convenience
}

// Sync types
export interface SyncItem {
    id: string
    client_id: string
    type: "project" | "task"
    slug?: string
    name?: string
    project_id?: string
    encrypted_data?: string // For projects
    encrypted_content?: string // For tasks
    status?: string
    priority?: number
    due_date?: string
    sync_version: number
    deleted: boolean
    client_updated_at?: string
}

export interface SyncPullResponse {
    items: SyncItem[]
    sync_version: number
}

export interface SyncPushRequest {
    items: SyncItem[]
}

export interface SyncPushResponse {
    updated: SyncItem[]
    conflicts?: ConflictItem[]
}

export interface ConflictItem {
    client_id: string
    type: string
    server_version: number
    server_data: SyncItem
    client_data: SyncItem
}

// Task type (decrypted, for UI)
export interface Task {
    id: string
    project_id: string
    content: string
    status: "process" | "done" | "ignore"
    priority: number
    due_date?: string
    sync_version: number
    deleted: boolean
}

// Project type (decrypted, for UI)
export interface Project {
    id: string
    slug: string
    name: string
    color?: string
    sync_version: number
    deleted: boolean
}

// Helper to get auth token
function getAuthToken(): string | null {
    if (typeof window === "undefined") return null
    return localStorage.getItem("token")
}

// Helper for authenticated requests
async function authFetch(url: string, options: RequestInit = {}): Promise<Response> {
    const token = getAuthToken()
    if (!token) {
        throw new Error("Not authenticated")
    }

    const headers = {
        ...options.headers,
        "Authorization": `Bearer ${token}`,
        "Content-Type": "application/json",
    }

    const res = await fetch(url, { ...options, headers })

    if (res.status === 401) {
        // Clear invalid token
        localStorage.removeItem("token")
        localStorage.removeItem("user_id")
        throw new Error("Session expired. Please login again.")
    }

    return res
}

// Auth endpoints
export async function sendMagicLink(email: string): Promise<MagicLinkResponse> {
    const res = await fetch(`${API_BASE}/magic-link`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
    })

    if (!res.ok) {
        const error = await res.json()
        throw new Error(error.error || "Failed to send magic link")
    }

    return res.json()
}

export async function verifyMagicLink(token: string): Promise<AuthResponse> {
    const res = await fetch(`${API_BASE}/magic-link/${token}`, {
        method: "GET",
    })

    if (!res.ok) {
        const error = await res.json()
        throw new Error(error.error || "Failed to verify magic link")
    }

    return res.json()
}

export async function login(email: string, password: string): Promise<AuthResponse> {
    const res = await fetch(`${API_BASE}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
    })

    if (!res.ok) {
        const error = await res.json()
        throw new Error(error.error || "Login failed")
    }

    return res.json()
}

export async function logout(): Promise<void> {
    try {
        await authFetch(`${API_BASE}/logout`, { method: "POST" })
    } finally {
        localStorage.removeItem("token")
        localStorage.removeItem("user_id")
        localStorage.removeItem("encryption_key")
        localStorage.removeItem("encryption_salt")
    }
}

// Sync endpoints
export async function syncPull(since: number = 0): Promise<SyncPullResponse> {
    const res = await authFetch(`${API_BASE}/sync?since=${since}`)

    if (!res.ok) {
        const error = await res.json()
        throw new Error(error.error || "Sync pull failed")
    }

    return res.json()
}

export async function syncPush(items: SyncItem[]): Promise<SyncPushResponse> {
    const res = await authFetch(`${API_BASE}/sync`, {
        method: "POST",
        body: JSON.stringify({ items }),
    })

    if (!res.ok) {
        const error = await res.json()
        throw new Error(error.error || "Sync push failed")
    }

    return res.json()
}

// User info
export async function getMe(): Promise<{ user_id: string; email: string }> {
    const res = await authFetch(`${API_BASE}/me`)

    if (!res.ok) {
        const error = await res.json()
        throw new Error(error.error || "Failed to get user info")
    }

    return res.json()
}

// Check if user is authenticated
export function isAuthenticated(): boolean {
    if (typeof window === "undefined") return false
    return !!localStorage.getItem("token")
}
