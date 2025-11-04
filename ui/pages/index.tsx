import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import ClicksChart from '../components/ClicksChart'

const API = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080'
const REDIRECT_BASE = process.env.NEXT_PUBLIC_REDIRECT_BASE || 'http://localhost:8085'

type LinkItem = {
  slug: string
  target_url: string
  is_active: boolean
  created_at: string
}

type Point = { day: string; total: number }

type Stats = { total?: number; points?: Point[] }

// Replaced inline SVG chart with ClicksChart (Recharts-based)

export default function Home() {
  const router = useRouter()
  const [token, setToken] = useState('')

  const [targetUrl, setTargetUrl] = useState('')
  const [customSlug, setCustomSlug] = useState('')
  const [links, setLinks] = useState<LinkItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [stats, setStats] = useState<Record<string, Stats>>({})
  const [editing, setEditing] = useState<Record<string, boolean>>({})
  const [editUrl, setEditUrl] = useState<Record<string, string>>({})
  const [activeTab, setActiveTab] = useState<'manage' | 'dashboard'>('manage')
  const [selectedSlug, setSelectedSlug] = useState<string>('')
  const [dashStats, setDashStats] = useState<Stats | null>(null)
  const [timeMode, setTimeMode] = useState<'relative' | 'absolute'>('relative')
  const [relativeRange, setRelativeRange] = useState<'5m' | '15m' | '1h' | '6h' | '24h' | '3d' | '7d' | '30d' | '90d'>('7d')
  const [absFrom, setAbsFrom] = useState<string>('')
  const [absTo, setAbsTo] = useState<string>('')
  const [resolution, setResolution] = useState<'auto' | 'day' | 'hour' | 'minute' | 'second'>('auto')

  useEffect(() => {
    const t = window.localStorage.getItem('token')
    if (t) setToken(t)
  }, [])

  useEffect(() => {
    if (token) {
      loadLinks(token)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])

  // When links are loaded and on Dashboard tab, auto-select first link if none selected
  useEffect(() => {
    if (activeTab !== 'dashboard') return
    if (selectedSlug) return
    if (links.length > 0) setSelectedSlug(links[0].slug)
  }, [activeTab, links, selectedSlug])

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

  async function patchLink(slug: string, body: any) {
    if (!token) return
    try {
      const res = await fetch(`${API}/api/v1/links/${slug}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body)
      })
      if (!res.ok) throw new Error(await res.text())
      await loadLinks()
    } catch (err) {
      setError('Update failed')
    }
  }

  async function toggleActive(slug: string, current: boolean) {
    await patchLink(slug, { is_active: !current })
  }

  async function updateUrl(slug: string) {
    const url = editUrl[slug]
    await patchLink(slug, { target_url: url })
    setEditing({ ...editing, [slug]: false })
  }

  async function loadStats(slug: string) {
    if (!token) return
    try {
      const res = await fetch(`${API}/api/v1/links/${slug}/stats?range=7d`, { headers: { Authorization: `Bearer ${token}` } })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setStats({ ...stats, [slug]: (data?.data || { total: 0, points: [] }) as Stats })
    } catch {
      setStats({ ...stats, [slug]: { total: 0, points: [] } })
    }
  }

  async function loadDashboardStats(sl: string, r: string) {
    if (!token || !sl) return
    try {
      const bucket = resolution === 'auto' ? 'day' : resolution
      const res = await fetch(`${API}/api/v1/links/${sl}/stats?bucket=${bucket}&range=${encodeURIComponent(r)}`, { headers: { Authorization: `Bearer ${token}` } })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setDashStats((data?.data || { total: 0, points: [] }) as Stats)
    } catch {
      setDashStats({ total: 0, points: [] })
    }
  }

  async function loadDashboardStatsFromTo(sl: string, from: string, to: string) {
    if (!token || !sl || !from || !to) return
    try {
      const bucket = resolution === 'auto' ? 'day' : resolution
      const res = await fetch(`${API}/api/v1/links/${sl}/stats?bucket=${bucket}&from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`, { headers: { Authorization: `Bearer ${token}` } })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setDashStats((data?.data || { total: 0, points: [] }) as Stats)
    } catch {
      setDashStats({ total: 0, points: [] })
    }
  }

  function computeAutoResolution(ms: number): 'day' | 'hour' | 'minute' | 'second' {
    const oneMinute = 60*1000
    const oneHour = 60*oneMinute
    const oneDay = 24*oneHour
    if (ms <= oneHour) return 'second'
    if (ms <= oneDay) return 'minute'
    if (ms <= 30*oneDay) return 'hour'
    return 'day'
  }

  // Auto-load dashboard stats when selections change
  useEffect(() => {
    if (activeTab !== 'dashboard') return
    if (!selectedSlug) return
    if (timeMode === 'relative') {
      const map: Record<string, number> = { '5m': 5*60*1000, '15m': 15*60*1000, '1h': 60*60*1000, '6h': 6*60*60*1000, '24h': 24*60*60*1000, '3d': 3*24*60*60*1000, '7d': 7*24*60*60*1000, '30d': 30*24*60*60*1000, '90d': 90*24*60*60*1000 }
      const ms = map[relativeRange]
      const bucket = resolution === 'auto' ? computeAutoResolution(ms) : resolution
      ;(async () => {
        try {
          const res = await fetch(`${API}/api/v1/links/${selectedSlug}/stats?bucket=${bucket}&range=${encodeURIComponent(relativeRange)}`, { headers: { Authorization: `Bearer ${token}` } })
          if (!res.ok) throw new Error(await res.text())
          const data = await res.json()
          setDashStats((data?.data || { total: 0, points: [] }) as Stats)
        } catch { setDashStats({ total: 0, points: [] }) }
      })()
    } else if (absFrom && absTo) {
      const fromD = new Date(absFrom)
      const toD = new Date(absTo)
      if (isNaN(fromD.getTime()) || isNaN(toD.getTime()) || toD < fromD) return
      const ms = toD.getTime() - fromD.getTime()
      const bucket = resolution === 'auto' ? computeAutoResolution(ms) : resolution
      const fromRFC = new Date(fromD.getTime() - (fromD.getTimezoneOffset()*60000)).toISOString()
      const toRFC = new Date(toD.getTime() - (toD.getTimezoneOffset()*60000)).toISOString()
      ;(async () => {
        try {
          const res = await fetch(`${API}/api/v1/links/${selectedSlug}/stats?bucket=${bucket}&from=${encodeURIComponent(fromRFC)}&to=${encodeURIComponent(toRFC)}`, { headers: { Authorization: `Bearer ${token}` } })
          if (!res.ok) throw new Error(await res.text())
          const data = await res.json()
          setDashStats((data?.data || { total: 0, points: [] }) as Stats)
        } catch { setDashStats({ total: 0, points: [] }) }
      })()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, selectedSlug, relativeRange, timeMode, absFrom, absTo, resolution])

  return (
    <div className="min-h-screen bg-gray-100">
      <header className="bg-white border-b">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <h1 className="text-xl font-semibold">Shortlink</h1>
          <div className="flex items-center gap-3">
            {!token ? (
              <button className="px-3 py-2 bg-black text-white rounded text-sm" onClick={()=> router.push('/login')}>Đăng nhập</button>
            ) : (
              <>
                <nav className="hidden md:flex items-center gap-2 mr-3">
                  <button
                    className={`px-3 py-1 rounded text-sm border ${activeTab==='manage' ? 'bg-gray-900 text-white border-gray-900' : 'bg-white'}`}
                    onClick={()=> setActiveTab('manage')}
                  >Tạo & Quản lý</button>
                  <button
                    className={`px-3 py-1 rounded text-sm border ${activeTab==='dashboard' ? 'bg-gray-900 text-white border-gray-900' : 'bg-white'}`}
                    onClick={()=> setActiveTab('dashboard')}
                  >Dashboard</button>
                </nav>
                <button className="px-2 py-1 border rounded text-sm" onClick={()=>{ setToken(''); window.localStorage.removeItem('token'); setLinks([]); setDashStats(null); }}>Logout</button>
              </>
            )}
          </div>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-6 space-y-6">
        {error && <div className="bg-red-50 text-red-700 px-3 py-2 rounded border border-red-200 text-sm">{error}</div>}

        {!token && (
          <section className="bg-white p-6 rounded shadow text-center">
            <h2 className="font-semibold text-lg">Chào mừng đến Shortlink</h2>
            <p className="text-sm text-gray-600 mt-1">Vui lòng đăng nhập để quản lý link rút gọn.</p>
            <button className="mt-4 px-4 py-2 bg-blue-600 text-white rounded" onClick={()=> router.push('/login')}>Đăng nhập</button>
          </section>
        )}

        {token && activeTab === 'manage' && (
          <>
            <section className="bg-white p-4 rounded shadow">
              <h2 className="font-semibold mb-3">Tạo link</h2>
              <form onSubmit={createLink} className="grid gap-2 md:grid-cols-3">
                <input className="border p-2 rounded col-span-2" placeholder="Target URL (https://...)" value={targetUrl} onChange={e=>setTargetUrl(e.target.value)} />
                <input className="border p-2 rounded" placeholder="Custom slug (optional)" value={customSlug} onChange={e=>setCustomSlug(e.target.value)} />
                <div className="md:col-span-3 flex items-center gap-3 mt-2">
                  <button className="px-3 py-2 bg-blue-600 text-white rounded" type="submit">Create</button>
                  <button type="button" className="text-sm underline" onClick={()=>loadLinks()} disabled={loading}>{loading ? 'Loading...' : 'Refresh list'}</button>
                </div>
              </form>
            </section>

            <section className="bg-white p-4 rounded shadow">
              <div className="flex items-center justify-between mb-2">
                <h2 className="font-semibold">Quản lý links</h2>
                <button className="text-sm underline" onClick={()=>loadLinks()} disabled={loading}>{loading ? 'Loading...' : 'Refresh'}</button>
              </div>
              <ul className="divide-y">
                {links.map((l)=> {
                  const st = stats[l.slug] as Stats
                  const pts = (st?.points || []) as Point[]
                  return (
                    <li key={l.slug} className="py-3">
                      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
                        <div>
                          <div className="font-mono text-sm">{l.slug}</div>
                          {!editing[l.slug] ? (
                            <div className="text-sm text-gray-600 break-all">{l.target_url}</div>
                          ) : (
                            <input className="border p-1 text-sm w-full md:w-[460px] rounded" value={editUrl[l.slug] ?? l.target_url} onChange={e=>setEditUrl({...editUrl, [l.slug]: e.target.value})} />
                          )}
                          <div className="space-x-3 mt-1">
                            <a className="text-blue-700 underline text-sm" href={`${REDIRECT_BASE}/${l.slug}`} target="_blank" rel="noreferrer">Open</a>
                            <button className="text-sm underline" onClick={()=> loadStats(l.slug)}>Stats (7d)</button>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          {!editing[l.slug] ? (
                            <>
                              <button className="text-sm px-2 py-1 border rounded" onClick={()=> setEditing({...editing, [l.slug]: true})}>Edit</button>
                              <button className="text-sm px-2 py-1 border rounded" onClick={()=> toggleActive(l.slug, l.is_active)}>{l.is_active ? 'Deactivate' : 'Activate'}</button>
                              <button className="text-sm px-2 py-1 border rounded text-red-600" onClick={()=> deleteLink(l.slug)}>Delete</button>
                            </>
                          ) : (
                            <>
                              <button className="text-sm px-2 py-1 border rounded" onClick={()=> updateUrl(l.slug)}>Save</button>
                              <button className="text-sm px-2 py-1 border rounded" onClick={()=> setEditing({...editing, [l.slug]: false})}>Cancel</button>
                            </>
                          )}
                        </div>
                      </div>
                      {st && pts.length > 0 && (
                        <div className="mt-3">
                          <ClicksChart points={pts as any} />
                          <div className="text-xs text-gray-600 mt-1">Total: {st.total || 0}</div>
                        </div>
                      )}
                    </li>
                  )})}
                {links.length === 0 && <li className="py-6 text-sm text-gray-500 text-center">No links yet</li>}
              </ul>
            </section>
          </>
        )}

        {token && activeTab === 'dashboard' && (
          <section className="bg-white p-4 rounded shadow">
            <div className="flex flex-col md:flex-row md:items-end md:justify-between gap-4 mb-4">
              <div className="flex-1 min-w-[220px]">
                <label className="block text-sm text-gray-700 mb-1">Chọn link</label>
                <select
                  className="w-full border rounded px-3 py-2"
                  value={selectedSlug}
                  onChange={(e)=> { setSelectedSlug(e.target.value); setDashStats(null); }}
                >
                  <option value="">-- Chọn --</option>
                  {links.map(l => (
                    <option key={l.slug} value={l.slug}>{l.slug}</option>
                  ))}
                </select>
              </div>
              <div className="flex-1 min-w-[320px]">
                <label className="block text-sm text-gray-700 mb-1">Thời gian</label>
                <div className="flex flex-wrap items-center gap-2">
                  <button type="button" className={`px-3 py-1 rounded border text-sm ${timeMode==='relative' ? 'bg-gray-900 text-white border-gray-900' : ''}`} onClick={()=>{ setTimeMode('relative'); setDashStats(null); }}>Tương đối</button>
                  <button type="button" className={`px-3 py-1 rounded border text-sm ${timeMode==='absolute' ? 'bg-gray-900 text-white border-gray-900' : ''}`} onClick={()=>{ setTimeMode('absolute'); setDashStats(null); }}>Tuyệt đối</button>
                  {timeMode==='relative' && (
                    <select className="border rounded px-3 py-2" value={relativeRange} onChange={(e)=> { setRelativeRange(e.target.value as any); setDashStats(null); }}>
                      <option value="5m">5 phút</option>
                      <option value="15m">15 phút</option>
                      <option value="1h">1 giờ</option>
                      <option value="6h">6 giờ</option>
                      <option value="24h">24 giờ</option>
                      <option value="3d">3 ngày</option>
                      <option value="7d">7 ngày</option>
                      <option value="30d">30 ngày</option>
                      <option value="90d">90 ngày</option>
                    </select>
                  )}
                </div>
              </div>
              <div>
                <label className="block text-sm text-gray-700 mb-1">Độ phân giải</label>
                <select className="border rounded px-3 py-2" value={resolution} onChange={(e)=> { setResolution(e.target.value as any); setDashStats(null); }}>
                  <option value="auto">Tự động</option>
                  <option value="day">Ngày</option>
                  <option value="hour">Giờ</option>
                  <option value="minute">Phút</option>
                  <option value="second">Giây</option>
                </select>
              </div>
              {timeMode === 'absolute' && (
                <div className="flex items-end gap-3">
                  <div>
                    <label className="block text-sm text-gray-700 mb-1">Từ</label>
                    <input type="datetime-local" className="border rounded px-3 py-2" value={absFrom} onChange={(e)=> { setAbsFrom(e.target.value); setDashStats(null); }} />
                  </div>
                  <div>
                    <label className="block text-sm text-gray-700 mb-1">Đến</label>
                    <input type="datetime-local" className="border rounded px-3 py-2" value={absTo} onChange={(e)=> { setAbsTo(e.target.value); setDashStats(null); }} />
                  </div>
                </div>
              )}
            </div>

            {!selectedSlug && <div className="text-sm text-gray-500">Hãy chọn một link để xem thống kê.</div>}
            {selectedSlug && dashStats && (
              <div>
                <ClicksChart points={(dashStats.points || []) as any} />
                <div className="text-xs text-gray-600 mt-1">Tổng lượt click: {dashStats.total || 0}</div>
              </div>
            )}
          </section>
        )}
      </main>
    </div>
  )
}
