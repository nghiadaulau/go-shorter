import { useEffect, useState } from 'react'

const API = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080'
const REDIRECT_BASE = process.env.NEXT_PUBLIC_REDIRECT_BASE || 'http://localhost:8082'

type LinkItem = {
  slug: string
  target_url: string
  is_active: boolean
  created_at: string
}

export default function Home() {
  const [email, setEmail] = useState('admin@example.com')
  const [password, setPassword] = useState('Admin@123')
  const [token, setToken] = useState('')

  const [targetUrl, setTargetUrl] = useState('')
  const [customSlug, setCustomSlug] = useState('')
  const [links, setLinks] = useState<LinkItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const t = window.localStorage.getItem('token')
    if (t) setToken(t)
  }, [])

  async function login(e: React.FormEvent) {
    e.preventDefault()
    setError('')
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
      setToken(t)
      window.localStorage.setItem('token', t)
      await loadLinks(t)
    } catch (err: any) {
      setError(err.message || 'Login failed')
    }
  }

  async function loadLinks(t?: string) {
    const tk = t || token
    if (!tk) return
    setLoading(true)
    setError('')
    try {
      const res = await fetch(`${API}/api/v1/links`, {
        headers: { Authorization: `Bearer ${tk}` },
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setLinks(data?.data || [])
    } catch (err: any) {
      setError('Load links failed')
    } finally {
      setLoading(false)
    }
  }

  async function createLink(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!token) return setError('Please login first')
    try {
      const res = await fetch(`${API}/api/v1/links`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ target_url: targetUrl, custom_slug: customSlug || undefined })
      })
      if (!res.ok) throw new Error(await res.text())
      setTargetUrl(''); setCustomSlug('')
      await loadLinks()
    } catch (err: any) {
      setError('Create link failed')
    }
  }

  async function deleteLink(slug: string) {
    if (!token) return
    try {
      const res = await fetch(`${API}/api/v1/links/${slug}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok && res.status !== 204) throw new Error(await res.text())
      await loadLinks()
    } catch (err) {
      setError('Delete failed')
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-3xl mx-auto space-y-6">
        <h1 className="text-2xl font-bold">Shortlink Dashboard</h1>

        {!token ? (
          <form onSubmit={login} className="grid gap-2 bg-white p-4 rounded shadow">
            <div className="font-semibold">Login</div>
            <input className="border p-2" placeholder="Email" value={email} onChange={e=>setEmail(e.target.value)} />
            <input className="border p-2" placeholder="Password" type="password" value={password} onChange={e=>setPassword(e.target.value)} />
            {error && <div className="text-red-600 text-sm">{error}</div>}
            <button className="px-3 py-2 bg-black text-white rounded" type="submit">Login</button>
          </form>
        ) : (
          <div className="flex items-center gap-3">
            <div className="text-sm text-gray-700">Logged in</div>
            <button className="px-2 py-1 border rounded" onClick={()=>{ setToken(''); window.localStorage.removeItem('token'); }}>Logout</button>
          </div>
        )}

        {token && (
          <>
            <form onSubmit={createLink} className="grid gap-2 bg-white p-4 rounded shadow">
              <div className="font-semibold">Create Link</div>
              <input className="border p-2" placeholder="Target URL (https://...)" value={targetUrl} onChange={e=>setTargetUrl(e.target.value)} />
              <input className="border p-2" placeholder="Custom slug (optional)" value={customSlug} onChange={e=>setCustomSlug(e.target.value)} />
              {error && <div className="text-red-600 text-sm">{error}</div>}
              <button className="px-3 py-2 bg-blue-600 text-white rounded" type="submit">Create</button>
            </form>

            <div className="bg-white p-4 rounded shadow">
              <div className="flex items-center justify-between">
                <div className="font-semibold">Links</div>
                <button className="text-sm underline" onClick={()=>loadLinks()} disabled={loading}>{loading ? 'Loading...' : 'Refresh'}</button>
              </div>
              <ul className="divide-y mt-3">
                {links.map((l)=> (
                  <li key={l.slug} className="py-3 flex items-center justify-between">
                    <div>
                      <div className="font-mono">{l.slug}</div>
                      <div className="text-sm text-gray-600">{l.target_url}</div>
                      <a className="text-blue-700 underline text-sm" href={`${REDIRECT_BASE}/${l.slug}`} target="_blank" rel="noreferrer">Open</a>
                    </div>
                    <button className="text-red-600 text-sm" onClick={()=>deleteLink(l.slug)}>Delete</button>
                  </li>
                ))}
                {links.length === 0 && <li className="py-3 text-sm text-gray-500">No links yet</li>}
              </ul>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
