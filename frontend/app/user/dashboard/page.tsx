'use client';

import { useEffect, useState } from 'react';
import { apiDelete, apiGet, apiPost } from '@/lib/api';
import Link from 'next/link';

const MODEL_OPTIONS = ['gpt-4o-mini','gpt-4o','gpt-4.1-mini','gpt-4.1','gpt-3.5-turbo','claude-3-5-sonnet','claude-3-5-haiku','claude-3-opus','gemini-1.5-pro','gemini-1.5-flash','gemini-2.5-flash','gemini-1.0-pro'];

export default function UserDashboard() {
  const [usage, setUsage] = useState<any[]>([]);
  const [keys, setKeys] = useState<any[]>([]);
  const [profile, setProfile] = useState<any | null>(null);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [keyName, setKeyName] = useState('');
  const [models, setModels] = useState<string[]>(['gpt-4o-mini']);

  async function refreshAll() {
    const token = localStorage.getItem('routerx_user_token') || '';
    try {
      const prof = await apiGet('/user/profile', token);
      setProfile(prof);
    } catch {}
    try {
      const list = await apiGet('/user/usage', token);
      setUsage(Array.isArray(list) ? list : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load usage');
    }
    try {
      const list = await apiGet('/user/api-keys', token);
      setKeys(Array.isArray(list) ? list : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load keys');
    }
  }

  useEffect(() => {
    refreshAll();
    const timer = setInterval(() => {
      refreshAll();
    }, 30000);
    return () => clearInterval(timer);
  }, []);

  async function createKey() {
    setStatus('');
    setError('');
    try {
      const token = localStorage.getItem('routerx_user_token') || '';
      const res = await apiPost('/user/api-keys', { name: keyName, allowed_models: models }, token);
      setKeys((prev) => [...prev, { key: res.key, name: keyName, allowed_models: models, created_at: res.created_at }]);
      setKeyName('');
      setStatus('API key created');
    } catch (err: any) {
      setError(err.message || 'Failed to create key');
    }
  }

  function toggleModel(m: string) {
    setModels((prev) => prev.includes(m) ? prev.filter((x) => x !== m) : [...prev, m]);
  }

  function maskKey(value: string) {
    if (!value) return '';
    if (value.length <= 10) return `${value.slice(0, 3)}•••${value.slice(-2)}`;
    return `${value.slice(0, 6)}•••${value.slice(-4)}`;
  }

  async function copyKey(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setStatus('API key copied');
    } catch {
      setError('Failed to copy key');
    }
  }

  async function deleteKey(value: string) {
    setError('');
    setStatus('');
    try {
      const token = localStorage.getItem('routerx_user_token') || '';
      await apiDelete(`/user/api-keys/${encodeURIComponent(value)}`, token);
      setKeys((prev) => prev.filter((k) => k.key !== value));
      setStatus('API key deleted');
    } catch (err: any) {
      setError(err.message || 'Failed to delete key');
    }
  }

  const usageDays = Array.from(new Set(usage.map((u) => new Date(u.day).toLocaleDateString()))).sort((a, b) => {
    const ad = new Date(a).getTime();
    const bd = new Date(b).getTime();
    return ad - bd;
  });

  const usageByModel = usage.reduce((acc: Record<string, Record<string, number>>, u) => {
    const day = new Date(u.day).toLocaleDateString();
    const model = u.model || 'unknown';
    const tokens = Number(u.tokens || 0);
    if (!acc[model]) acc[model] = {};
    acc[model][day] = (acc[model][day] || 0) + tokens;
    return acc;
  }, {});

  const chartModels = Object.keys(usageByModel);
  const chartData = chartModels.map((m) => usageDays.map((d) => usageByModel[m]?.[d] || 0));
  const maxTokens = Math.max(1, ...chartData.flat());
  const chartColors = ['#0f172a', '#2563eb', '#16a34a', '#dc2626', '#7c3aed', '#ea580c'];

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">User Dashboard</h1>
            <p className="text-sm text-black/60">Your API keys, balance, and usage only.</p>
          </div>
          <div className="text-sm">
            <div className="flex items-center gap-3">
              <button className="text-xs underline" onClick={refreshAll}>Refresh</button>
              <div className="w-9 h-9 rounded-full bg-ink text-white flex items-center justify-center text-sm">
                {(profile?.username || 'U').slice(0,1).toUpperCase()}
              </div>
              <span className="text-sm">{profile?.username || 'User'}</span>
              <Link href="/login" className="underline text-sm">Logout</Link>
            </div>
          </div>
        </div>

        {profile && (
          <section className="card p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-black/60">Workspace</p>
                <p className="text-lg font-semibold">{profile.name}</p>
              </div>
              <div>
                <p className="text-xs text-black/60">Balance (USD)</p>
                <p className="text-lg font-semibold">${Number(profile.balance_usd || 0).toFixed(2)}</p>
              </div>
            </div>
          </section>
        )}

        {error && <p className="text-red-500">{error}</p>}
        {status && <p className="text-green-600">{status}</p>}

        <section className="card p-4">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">API Keys</h2>
            <button className="px-3 py-2 rounded-lg bg-ink text-white" onClick={createKey}>New Key</button>
          </div>
          <div className="mt-3 space-y-4">
            <div className="space-y-2 text-sm">
              <label className="block">
                Key Name
                <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" value={keyName} onChange={(e) => setKeyName(e.target.value)} placeholder="My App" />
              </label>
              <div>
                <p className="text-xs text-black/60 mb-2">Allowed Models</p>
                <div className="flex flex-wrap gap-2">
                  {MODEL_OPTIONS.map((m) => (
                    <button key={m} type="button" onClick={() => toggleModel(m)} className={`px-2 py-1 rounded-full border text-xs ${models.includes(m) ? 'bg-ink text-white' : 'border-black/10'}`}>
                      {m}
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div>
              {!keys.length && <p className="text-black/50 text-sm">No keys yet.</p>}
              {!!keys.length && (
                <div className="border rounded-lg overflow-hidden text-sm">
                  <div className="grid grid-cols-12 gap-2 px-3 py-2 bg-black/5 text-xs font-semibold">
                    <div className="col-span-3">Name</div>
                    <div className="col-span-3">Key</div>
                    <div className="col-span-4">Models</div>
                    <div className="col-span-2">Created</div>
                  </div>
                  {keys.map((k, idx) => (
                    <div key={idx} className="grid grid-cols-12 gap-2 px-3 py-2 border-t items-center">
                      <div className="col-span-3 font-semibold">{k.name || 'Unnamed Key'}</div>
                      <div className="col-span-3 flex items-center gap-2">
                        <span className="font-mono text-xs">{maskKey(k.key)}</span>
                        <button className="text-xs underline" onClick={() => copyKey(k.key)}>Copy</button>
                        <button className="text-xs text-red-600 underline" onClick={() => deleteKey(k.key)}>Delete</button>
                      </div>
                      <div className="col-span-4 text-xs text-black/70">{(k.allowed_models || []).join(', ') || 'all'}</div>
                      <div className="col-span-2 text-xs text-black/70">{k.created_at ? new Date(k.created_at).toLocaleDateString() : '-'}</div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="card p-4">
          <h2 className="font-semibold">Recent Usage</h2>
          {!usage.length && <p className="text-sm text-black/50 mt-2">No usage yet.</p>}
          {usage.length > 0 && (
            <div className="mt-3">
              <div className="flex flex-wrap gap-3 text-xs text-black/60 mb-3">
                {chartModels.map((m, i) => (
                  <div key={m} className="flex items-center gap-2">
                    <span className="inline-block w-3 h-3 rounded-sm" style={{ background: chartColors[i % chartColors.length] }} />
                    <span>{m}</span>
                  </div>
                ))}
              </div>
              <div className="w-full overflow-x-auto">
                <svg viewBox="0 0 800 260" className="w-full min-w-[560px]">
                  <rect x="0" y="0" width="800" height="260" rx="16" fill="#fff" stroke="rgba(0,0,0,0.08)" />
                  {[0.25, 0.5, 0.75, 1].map((p) => {
                    const y = 220 - p * 180;
                    return <line key={p} x1="40" y1={y} x2="770" y2={y} stroke="rgba(0,0,0,0.06)" strokeWidth="1" />;
                  })}
                  {usageDays.map((d, i) => {
                    const x = 40 + (i * (730 / Math.max(1, usageDays.length - 1)));
                    return (
                      <text key={d} x={x} y="245" textAnchor="middle" fontSize="10" fill="rgba(0,0,0,0.45)">{d}</text>
                    );
                  })}
                  {chartData.map((series, si) => {
                    const points = series.map((v, i) => {
                      const x = 40 + (i * (730 / Math.max(1, usageDays.length - 1)));
                      const y = 220 - (v / maxTokens) * 180;
                      return `${x},${y}`;
                    }).join(' ');
                    return <polyline key={si} fill="none" stroke={chartColors[si % chartColors.length]} strokeWidth="2.2" points={points} />;
                  })}
                </svg>
              </div>
              <p className="text-xs text-black/50 mt-2">Y-axis: tokens/day (stacked per model line). X-axis: day.</p>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}
