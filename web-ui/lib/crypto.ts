/**
 * Crypto utilities for AES-256-GCM encryption/decryption
 * Compatible with Go backend encryption (internal/sync/crypto.go)
 */

const KEY_SIZE = 32 // AES-256
const NONCE_SIZE = 12 // GCM standard nonce size
const SALT_SIZE = 16
const PBKDF2_ITERATIONS = 100000

/**
 * Derive encryption key from password and salt using PBKDF2
 */
export async function deriveKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
    const encoder = new TextEncoder()
    const passwordKey = await crypto.subtle.importKey(
        "raw",
        encoder.encode(password),
        "PBKDF2",
        false,
        ["deriveKey"]
    )

    return crypto.subtle.deriveKey(
        {
            name: "PBKDF2",
            salt: salt as BufferSource,
            iterations: PBKDF2_ITERATIONS,
            hash: "SHA-256",
        },
        passwordKey,
        { name: "AES-GCM", length: KEY_SIZE * 8 },
        true, // extractable for storage
        ["encrypt", "decrypt"]
    )
}

/**
 * Generate random salt
 */
export function generateSalt(): Uint8Array {
    return crypto.getRandomValues(new Uint8Array(SALT_SIZE))
}

/**
 * Encrypt data using AES-256-GCM
 * Returns base64-encoded string (nonce + ciphertext)
 */
export async function encrypt(key: CryptoKey, plaintext: string): Promise<string> {
    const encoder = new TextEncoder()
    const data = encoder.encode(plaintext)
    const nonce = crypto.getRandomValues(new Uint8Array(NONCE_SIZE))

    const ciphertext = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv: nonce },
        key,
        data
    )

    // Combine nonce + ciphertext
    const combined = new Uint8Array(nonce.length + ciphertext.byteLength)
    combined.set(nonce)
    combined.set(new Uint8Array(ciphertext), nonce.length)

    return btoa(String.fromCharCode(...combined))
}

/**
 * Decrypt data using AES-256-GCM
 * Input is base64-encoded string (nonce + ciphertext)
 */
export async function decrypt(key: CryptoKey, encrypted: string): Promise<string> {
    // Decode base64
    const combined = Uint8Array.from(atob(encrypted), c => c.charCodeAt(0))

    if (combined.length < NONCE_SIZE) {
        throw new Error("Ciphertext too short")
    }

    const nonce = combined.slice(0, NONCE_SIZE)
    const ciphertext = combined.slice(NONCE_SIZE)

    const decrypted = await crypto.subtle.decrypt(
        { name: "AES-GCM", iv: nonce },
        key,
        ciphertext
    )

    const decoder = new TextDecoder()
    return decoder.decode(decrypted)
}

/**
 * Export key to base64 for storage
 */
export async function exportKey(key: CryptoKey): Promise<string> {
    const raw = await crypto.subtle.exportKey("raw", key)
    return btoa(String.fromCharCode(...new Uint8Array(raw)))
}

/**
 * Import key from base64 storage
 */
export async function importKey(base64Key: string): Promise<CryptoKey> {
    const raw = Uint8Array.from(atob(base64Key), c => c.charCodeAt(0))
    return crypto.subtle.importKey(
        "raw",
        raw,
        { name: "AES-GCM", length: KEY_SIZE * 8 },
        true,
        ["encrypt", "decrypt"]
    )
}

/**
 * Store encryption credentials in localStorage
 */
export async function storeEncryptionKey(password: string): Promise<void> {
    const salt = generateSalt()
    const key = await deriveKey(password, salt)
    const exportedKey = await exportKey(key)

    localStorage.setItem("encryption_key", exportedKey)
    localStorage.setItem("encryption_salt", btoa(String.fromCharCode(...salt)))
}

/**
 * Get encryption key from localStorage
 */
export async function getStoredKey(): Promise<CryptoKey | null> {
    const exportedKey = localStorage.getItem("encryption_key")
    if (!exportedKey) return null

    try {
        return await importKey(exportedKey)
    } catch {
        return null
    }
}

/**
 * Get stored salt from localStorage
 */
export function getStoredSalt(): Uint8Array | null {
    const saltBase64 = localStorage.getItem("encryption_salt")
    if (!saltBase64) return null

    try {
        return Uint8Array.from(atob(saltBase64), c => c.charCodeAt(0))
    } catch {
        return null
    }
}

/**
 * Check if encryption is set up
 */
export function hasEncryptionKey(): boolean {
    return !!localStorage.getItem("encryption_key")
}

/**
 * Clear encryption credentials
 */
export function clearEncryptionKey(): void {
    localStorage.removeItem("encryption_key")
    localStorage.removeItem("encryption_salt")
}

// Type for encrypted task content
export interface EncryptedTaskContent {
    content: string
}

// Type for encrypted project data
export interface EncryptedProjectData {
    name: string
    slug: string
    color?: string
}

/**
 * Decrypt task content
 * Handles both plain base64 JSON (from TUI) and AES-GCM encrypted data
 */
export async function decryptTaskContent(
    key: CryptoKey,
    encryptedContent: string
): Promise<EncryptedTaskContent> {
    // First try plain base64 JSON (TUI format)
    try {
        const plainJson = atob(encryptedContent)
        const parsed = JSON.parse(plainJson)
        if (parsed && parsed.content !== undefined) {
            return parsed
        }
    } catch {
        // Not plain JSON, try decryption
    }

    // Try AES-GCM decryption (web UI format)
    const json = await decrypt(key, encryptedContent)
    return JSON.parse(json)
}

/**
 * Encrypt task content
 * Uses plain base64 JSON format (compatible with TUI)
 */
export async function encryptTaskContent(
    _key: CryptoKey,
    content: string
): Promise<string> {
    const data: EncryptedTaskContent = { content }
    // Use plain base64 JSON for TUI compatibility
    return btoa(JSON.stringify(data))
}

/**
 * Decrypt project data
 * Handles both plain base64 JSON (from TUI) and AES-GCM encrypted data
 */
export async function decryptProjectData(
    key: CryptoKey,
    encryptedData: string
): Promise<EncryptedProjectData> {
    // First try plain base64 JSON (TUI format)
    try {
        const plainJson = atob(encryptedData)
        const parsed = JSON.parse(plainJson)
        if (parsed && (parsed.name || parsed.color)) {
            return parsed
        }
    } catch {
        // Not plain JSON, try decryption
    }

    // Try AES-GCM decryption (web UI format)
    const json = await decrypt(key, encryptedData)
    return JSON.parse(json)
}

/**
 * Encrypt project data
 * Uses plain base64 JSON format (compatible with TUI)
 */
export async function encryptProjectData(
    _key: CryptoKey,
    data: EncryptedProjectData
): Promise<string> {
    // Use plain base64 JSON for TUI compatibility
    return btoa(JSON.stringify(data))
}
