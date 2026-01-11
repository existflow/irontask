"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { sendMagicLink, verifyMagicLink } from "@/lib/api"
import { storeEncryptionKey } from "@/lib/crypto"
import { toast } from "sonner"
import { Loader2, Mail } from "lucide-react"

export default function LoginPage() {
  const [email, setEmail] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [magicLinkToken, setMagicLinkToken] = useState<string | null>(null)

  const router = useRouter()

  const handleSendMagicLink = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!email) {
      toast.error("Please enter your email")
      return
    }

    setIsLoading(true)
    try {
      await sendMagicLink(email)
      setMagicLinkToken("") // Show token input (empty for user to paste)
      toast.success("Magic link sent!")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to send link")
    } finally {
      setIsLoading(false)
    }
  }

  const handleVerifyToken = async () => {
    if (!magicLinkToken?.trim()) return

    setIsLoading(true)
    try {
      const auth = await verifyMagicLink(magicLinkToken)
      localStorage.setItem("token", auth.token)
      localStorage.setItem("user_id", auth.user_id)

      // Set up encryption key using email as password (simplified for now)
      await storeEncryptionKey(email)

      toast.success("Logged in!")
      router.push("/dashboard")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Verification failed")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <main className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900">Irontask</h1>
          <p className="text-gray-500 mt-2">Task management made simple</p>
        </div>

        <div className="bg-white rounded-lg shadow-sm border p-6">
          {magicLinkToken !== null ? (
            <div className="space-y-4">
              <div className="p-4 bg-green-50 border border-green-200 rounded-lg text-center">
                <Mail className="w-8 h-8 text-green-600 mx-auto mb-2" />
                <p className="font-medium text-green-800">Check your email</p>
                <p className="text-sm text-green-600 mt-1">We sent a login link to {email}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  Paste token from email
                </label>
                <input
                  type="text"
                  value={magicLinkToken}
                  onChange={(e) => setMagicLinkToken(e.target.value)}
                  placeholder="Paste your token here..."
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent font-mono text-sm"
                />
              </div>

              <button
                onClick={handleVerifyToken}
                disabled={isLoading || !magicLinkToken}
                className="w-full bg-gray-900 hover:bg-gray-800 disabled:bg-gray-400 text-white py-2.5 rounded-lg transition-colors flex items-center justify-center gap-2"
              >
                {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
                {isLoading ? "Verifying..." : "Login"}
              </button>

              <button
                onClick={() => setMagicLinkToken(null)}
                className="w-full text-sm text-gray-500 hover:text-gray-700"
              >
                Use different email
              </button>
            </div>
          ) : (
            <form onSubmit={handleSendMagicLink} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  Email
                </label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  required
                  autoFocus
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
                />
              </div>

              <button
                type="submit"
                disabled={isLoading}
                className="w-full bg-gray-900 hover:bg-gray-800 text-white py-2.5 rounded-lg transition-colors flex items-center justify-center gap-2"
              >
                {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Mail className="w-4 h-4" />}
                {isLoading ? "Sending..." : "Send Login Link"}
              </button>
            </form>
          )}
        </div>

        <p className="text-center text-sm text-gray-400 mt-6">
          We'll send you a magic link to sign in
        </p>
      </div>
    </main>
  )
}
