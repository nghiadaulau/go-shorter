import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'

const API = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080'

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = useState('admin@example.com')
  const [password, setPassword] = useState('Admin@123')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const t = window.localStorage.getItem('token')
    if (t) router.replace('/')
  }, [router])

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await fetch(`${API}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
      if (!res.ok) throw new Error('Invalid credentials')
      const data = await res.json()
      const t = data?.data?.access_token
      if (!t) throw new Error('No token returned')
      window.localStorage.setItem('token', t)
      router.replace('/')
    } catch (err: any) {
      setError(err?.message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-xl shadow-md p-6">
        <div className="mb-4 text-center">
          <h1 className="text-2xl font-semibold">Đăng nhập</h1>
          <p className="text-sm text-gray-600 mt-1">Truy cập trang quản trị Shortlink</p>
        </div>
        {error && (
          <div className="mb-3 bg-red-50 text-red-700 px-3 py-2 rounded border border-red-200 text-sm">{error}</div>
        )}
        <form onSubmit={handleLogin} className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input
              className="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Mật khẩu</label>
            <input
              className="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              type="password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>
          <button
            type="submit"
            className="w-full bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 font-medium transition"
            disabled={loading}
          >
            {loading ? 'Đang đăng nhập...' : 'Đăng nhập'}
          </button>
        </form>
        <div className="text-center mt-4">
          <button
            className="text-sm text-gray-600 underline"
            onClick={() => router.push('/')}
            type="button"
          >
            Quay lại trang chủ
          </button>
        </div>
      </div>
    </div>
  )
}


