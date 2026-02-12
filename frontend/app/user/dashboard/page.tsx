'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { apiDelete, apiGet, apiPost } from '@/lib/api';
import {
  BarChart, Bar, LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer,
  CartesianGrid, Cell
} from 'recharts';

const MODEL_COLORS = ['#0f172a', '#2563eb', '#16a34a', '#dc2626', '#7c3aed', '#ea580c', '#06b6d4', '#ec4899'];

// ---- Sub-components ----

function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse bg-black/5 rounded ${className}`} />;
}

function AccountCard({ profile, totalCost, onTopup }: { profile: any; totalCost: number; onTopup: () => void }) {
  return (
    <div className="card p-5">
      <div className="flex items-center justify-between mb-3">
        <div>
          <p className="text-xs text-black/50 uppercase tracking-wide">Account</p>
          <p className="text-sm text-black/60 mt-0.5">{profile?.name || 'Workspace'}</p>
        </div>
        <button onClick={onTopup} className="px-3 py-1.5 text-xs rounded-lg border border-black/10 hover:bg-black/5">
          Add Funds
        </button>
      </div>
      <div>
        <p className="text-xs text-black/50">Current Balance</p>
        <p className={`text-2xl font-semibold ${Number(profile?.balance_usd || 0) <= 0 ? 'text-red-500' : ''}`}>
          ${Number(profile?.balance_usd || 0).toFixed(4)}
        </p>
        <p className="text-xs text-black/40 mt-2">Total Spend: ${totalCost.toFixed(2)}</p>
      </div>
    </div>
  );
}

function StatsCard({ title, items }: { title: string; items: { label: string; value: string | number; series: number[]; color: string }[] }) {
  return (
    <div className="card p-5">
      <p className="text-xs text-black/50 uppercase tracking-wide mb-3">{title}</p>
      {items.map((item) => (
        <div key={item.label} className="flex items-center justify-between mb-3 last:mb-0">
          <div>
            <p className="text-xs text-black/50">{item.label}</p>
            <p className="text-xl font-semibold">{item.value}</p>
          </div>
          <div className="w-[120px] h-[36px]">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={item.series.map((v, i) => ({ i, v }))}>
                <Line type="monotone" dataKey="v" stroke={item.color} strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      ))}
    </div>
  );
}

function ModelAnalytics({ data }: { data: { model: string; cost: number; tokens: number }[] }) {
  if (!data.length) return <p className="text-sm text-black/40">No usage data yet.</p>;
  const maxCost = Math.max(1, ...data.map((d) => d.cost));
  return (
    <div className="space-y-3">
      {data.map((d, i) => (
        <div key={d.model} className="flex items-center gap-3">
          <div className="w-36 text-xs truncate text-black/70 font-medium">{d.model}</div>
          <div className="flex-1 h-4 bg-black/[0.04] rounded-full overflow-hidden">
            <div
              className="h-full rounded-full transition-all"
              style={{ width: `${(d.cost / maxCost) * 100}%`, background: MODEL_COLORS[i % MODEL_COLORS.length] }}
            />
          </div>
          <div className="w-24 text-xs text-black/60 text-right">
            ${d.cost.toFixed(4)} · {d.tokens.toLocaleString()} tok
          </div>
        </div>
      ))}
    </div>
  );
}

function APIKeysSection({ keys, onCopy, onDelete, onCreate, keyName, setKeyName, models, toggleModel, modelOptions }: any) {
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-3 items-end">
        <div className="flex-1 min-w-[200px]">
          <label className="text-sm font-medium block mb-1">Key Name</label>
          <input
            className="w-full border border-black/10 rounded-lg px-3 py-2 text-sm"
            value={keyName}
            onChange={(e: any) => setKeyName(e.target.value)}
            placeholder="My App"
          />
        </div>
        <button onClick={onCreate} className="px-4 py-2 rounded-lg bg-black text-white text-sm hover:bg-black/80">
          Create Key
        </button>
      </div>
      <div>
        <p className="text-xs text-black/50 mb-2">Allowed Models</p>
        <div className="flex flex-wrap gap-1.5">
          {modelOptions.map((m: string) => (
            <button
              key={m}
              type="button"
              onClick={() => toggleModel(m)}
              className={`px-2.5 py-1 rounded-full border text-xs transition-colors ${
                models.includes(m) ? 'bg-black text-white border-black' : 'border-black/10 hover:bg-black/5'
              }`}
            >
              {m}
            </button>
          ))}
        </div>
      </div>
      {!keys.length ? (
        <p className="text-black/40 text-sm">No API keys yet.</p>
      ) : (
        <div className="border rounded-lg overflow-hidden text-sm">
          <table className="w-full">
            <thead>
              <tr className="bg-black/[0.03] text-xs font-medium text-black/60 uppercase tracking-wide">
                <th className="px-4 py-2.5 text-left">Name</th>
                <th className="px-4 py-2.5 text-left">Key</th>
                <th className="px-4 py-2.5 text-left">Models</th>
                <th className="px-4 py-2.5 text-left">Created</th>
                <th className="px-4 py-2.5 text-center w-24">Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((k: any, idx: number) => (
                <tr key={idx} className="border-t border-black/5 hover:bg-black/[0.02]">
                  <td className="px-4 py-2.5 font-medium">{k.name || 'Unnamed Key'}</td>
                  <td className="px-4 py-2.5">
                    <span className="font-mono text-xs text-black/60">{maskKey(k.key)}</span>
                  </td>
                  <td className="px-4 py-2.5 text-xs text-black/60 max-w-[200px] truncate">
                    {(k.allowed_models || []).join(', ') || 'All models'}
                  </td>
                  <td className="px-4 py-2.5 text-xs text-black/50">
                    {k.created_at ? new Date(k.created_at).toLocaleDateString() : '-'}
                  </td>
                  <td className="px-4 py-2.5 text-center">
                    <button className="text-xs text-blue-600 hover:underline mr-2" onClick={() => onCopy(k.key)}>Copy</button>
                    <button className="text-xs text-red-500 hover:underline" onClick={() => onDelete(k.key)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function maskKey(value: string) {
  if (!value) return '';
  if (value.length <= 10) return `${value.slice(0, 3)}****${value.slice(-2)}`;
  return `${value.slice(0, 6)}****${value.slice(-4)}`;
}

// ---- Main component ----

const MODEL_OPTIONS = [
  'gpt-4o-mini', 'gpt-4o', 'gpt-4.1-mini', 'gpt-4.1', 'gpt-3.5-turbo',
  'claude-3-5-sonnet', 'claude-3-5-haiku', 'claude-3-opus',
  'gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-2.5-flash', 'gemini-1.0-pro'
];

export default function UserDashboard() {
  const [usage, setUsage] = useState<any[]>([]);
  const [keys, setKeys] = useState<any[]>([]);
  const [profile, setProfile] = useState<any>(null);
  const [summary, setSummary] = useState<any>(null);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(true);
  const [keyName, setKeyName] = useState('');
  const [models, setModels] = useState<string[]>(['gpt-4o-mini']);
  const [showTopup, setShowTopup] = useState(false);
  const [topupAmount, setTopupAmount] = useState('');
  const [topupBusy, setTopupBusy] = useState(false);

  async function refreshAll() {
    const token = localStorage.getItem('routerx_user_token') || '';
    setLoading(true);
    try {
      const [prof, usg, summ, k] = await Promise.all([
        apiGet('/user/profile', token).catch(() => null),
        apiGet('/user/usage', token).catch(() => []),
        apiGet('/user/summary', token).catch(() => null),
        apiGet('/user/api-keys', token).catch(() => [])
      ]);
      setProfile(prof);
      setUsage(Array.isArray(usg) ? usg : []);
      setSummary(summ);
      setKeys(Array.isArray(k) ? k : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refreshAll();
    const timer = setInterval(refreshAll, 30000);
    return () => clearInterval(timer);
  }, []);

  async function createKey() {
    setStatus(''); setError('');
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

  async function copyKey(value: string) {
    try { await navigator.clipboard.writeText(value); setStatus('API key copied'); } catch { setError('Failed to copy'); }
  }

  async function deleteKey(value: string) {
    setError(''); setStatus('');
    try {
      const token = localStorage.getItem('routerx_user_token') || '';
      await apiDelete(`/user/api-keys/${encodeURIComponent(value)}`, token);
      setKeys((prev) => prev.filter((k) => k.key !== value));
      setStatus('API key deleted');
    } catch (err: any) {
      setError(err.message || 'Failed to delete');
    }
  }

  async function submitTopup() {
    setError(''); setStatus('');
    const amount = Number(topupAmount);
    if (!Number.isFinite(amount) || amount <= 0) { setError('Enter a positive amount'); return; }
    setTopupBusy(true);
    try {
      const token = localStorage.getItem('routerx_user_token') || '';
      const res = await apiPost('/user/topup', { amount_usd: amount }, token);
      setProfile((prev: any) => prev ? { ...prev, balance_usd: res.balance_usd } : prev);
      setStatus('Balance updated');
      setTopupAmount('');
      setShowTopup(false);
    } catch (err: any) {
      setError(err.message || 'Failed');
    } finally {
      setTopupBusy(false);
    }
  }

  function logout() {
    localStorage.removeItem('routerx_user_token');
    window.location.href = '/login';
  }

  // Derived data
  const totalCost = Number(summary?.total_cost_usd || 0);
  const totalTokens = Number(summary?.total_tokens || 0);
  const totalRequests = Number(summary?.total_requests || 0);
  const dailySummary = Array.isArray(summary?.daily) ? summary.daily : [];
  const recentSummary = Array.isArray(summary?.recent) ? summary.recent : [];
  const requestSeries = recentSummary.map((d: any) => Number(d.requests || 0));
  const costSeries = recentSummary.map((d: any) => Number(d.cost_usd || 0));
  const tokenSeries = recentSummary.map((d: any) => Number(d.tokens || 0));

  // Model analytics
  const modelAgg = usage.reduce((acc: Record<string, { cost: number; tokens: number }>, u) => {
    const m = u.model || 'unknown';
    if (!acc[m]) acc[m] = { cost: 0, tokens: 0 };
    acc[m].cost += Number(u.cost_usd || 0);
    acc[m].tokens += Number(u.tokens || 0);
    return acc;
  }, {});
  const modelData = Object.entries(modelAgg)
    .filter(([, v]) => v.cost > 0 || v.tokens > 0)
    .sort((a, b) => b[1].cost - a[1].cost)
    .slice(0, 8)
    .map(([model, v]) => ({ model, cost: v.cost, tokens: v.tokens }));

  // Recent stacked bar chart data
  const recentModelRows = Array.isArray(summary?.recent_models) ? summary.recent_models : [];
  const recentModels = Array.from(new Set(recentModelRows.map((r: any) => r.model))) as string[];
  const recentChartData = recentSummary.map((bucket: any) => {
    const bucketTime = new Date(bucket.day).getTime();
    const point: any = { time: new Date(bucket.day).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) };
    recentModels.forEach((model) => {
      const row = recentModelRows.find((r: any) => r.model === model && new Date(r.bucket).getTime() === bucketTime);
      point[model] = row ? Number(row.tokens || 0) : 0;
    });
    return point;
  });

  if (loading) {
    return (
      <main className="min-h-screen p-8">
        <div className="max-w-6xl mx-auto space-y-6">
          <Skeleton className="h-10 w-48" />
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Skeleton className="h-36" /><Skeleton className="h-36" /><Skeleton className="h-36" />
          </div>
          <Skeleton className="h-48" />
          <Skeleton className="h-64" />
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Dashboard</h1>
            <p className="text-sm text-black/50">Your API usage, keys, and billing</p>
          </div>
          <div className="flex items-center gap-3">
            <button className="text-xs px-3 py-1.5 rounded-lg border border-black/10 hover:bg-black/5" onClick={refreshAll}>Refresh</button>
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-full bg-black text-white flex items-center justify-center text-sm font-medium">
                {(profile?.username || 'U').slice(0, 1).toUpperCase()}
              </div>
              <span className="text-sm font-medium">{profile?.username || 'User'}</span>
            </div>
            <button onClick={logout} className="text-sm text-black/50 hover:text-black">Logout</button>
          </div>
        </div>

        {/* Notifications */}
        {error && (
          <div className="card p-3 border-red-200 bg-red-50 text-sm text-red-600 flex items-center justify-between">
            <span>{error}</span>
            <button onClick={() => setError('')} className="text-red-400">✕</button>
          </div>
        )}
        {status && (
          <div className="card p-3 border-emerald-200 bg-emerald-50 text-sm text-emerald-700 flex items-center justify-between">
            <span>{status}</span>
            <button onClick={() => setStatus('')} className="text-emerald-400">✕</button>
          </div>
        )}

        {/* Stats Row */}
        <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <AccountCard profile={profile} totalCost={totalCost} onTopup={() => setShowTopup(true)} />
          <StatsCard
            title="Usage Statistics"
            items={[
              { label: 'Total Requests', value: totalRequests.toLocaleString(), series: requestSeries, color: '#16a34a' },
              { label: 'Data Points', value: `${dailySummary.length} days`, series: costSeries, color: '#2563eb' }
            ]}
          />
          <StatsCard
            title="Resource Usage"
            items={[
              { label: 'Total Cost', value: `$${totalCost.toFixed(2)}`, series: costSeries, color: '#f59e0b' },
              { label: 'Total Tokens', value: totalTokens.toLocaleString(), series: tokenSeries, color: '#ec4899' }
            ]}
          />
        </section>

        {/* Model Analytics */}
        <section className="card p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">Model Analytics</h2>
            <span className="text-xs text-black/40">Cost Distribution</span>
          </div>
          <ModelAnalytics data={modelData} />
        </section>

        {/* API Keys */}
        <section className="card p-5">
          <h2 className="text-lg font-semibold mb-4">API Keys</h2>
          <APIKeysSection
            keys={keys}
            onCopy={copyKey}
            onDelete={deleteKey}
            onCreate={createKey}
            keyName={keyName}
            setKeyName={setKeyName}
            models={models}
            toggleModel={toggleModel}
            modelOptions={MODEL_OPTIONS}
          />
        </section>

        {/* Recent Usage Chart */}
        <section className="card p-5">
          <h2 className="text-lg font-semibold mb-2">Recent Usage</h2>
          <p className="text-xs text-black/40 mb-4">Tokens per model in 3-hour buckets (last 24h)</p>
          {!recentSummary.length ? (
            <p className="text-sm text-black/40">No recent usage data.</p>
          ) : (
            <>
              <div className="flex flex-wrap gap-3 text-xs text-black/60 mb-3">
                {recentModels.map((m, i) => (
                  <div key={m} className="flex items-center gap-1.5">
                    <span className="inline-block w-3 h-3 rounded-sm" style={{ background: MODEL_COLORS[i % MODEL_COLORS.length] }} />
                    <span>{m}</span>
                  </div>
                ))}
              </div>
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={recentChartData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                    <XAxis dataKey="time" tick={{ fontSize: 11 }} />
                    <YAxis tick={{ fontSize: 11 }} />
                    <Tooltip contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb', fontSize: '12px' }} />
                    {recentModels.map((model, i) => (
                      <Bar key={model} dataKey={model} stackId="a" fill={MODEL_COLORS[i % MODEL_COLORS.length]} radius={i === recentModels.length - 1 ? [4, 4, 0, 0] : [0, 0, 0, 0]} />
                    ))}
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </>
          )}
        </section>
      </div>

      {/* Top-up Modal */}
      {showTopup && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 max-w-sm w-full mx-4">
            <h3 className="font-semibold text-lg mb-4">Add Funds</h3>
            <label className="text-sm font-medium block mb-1">Amount (USD)</label>
            <input
              type="number"
              step="0.01"
              className="w-full border border-black/10 rounded-lg px-3 py-2 text-sm"
              value={topupAmount}
              onChange={(e) => setTopupAmount(e.target.value)}
              placeholder="10.00"
              autoFocus
            />
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowTopup(false)} className="px-4 py-2 text-sm rounded-lg border border-black/10 hover:bg-black/5">
                Cancel
              </button>
              <button
                onClick={submitTopup}
                disabled={topupBusy}
                className="px-4 py-2 text-sm rounded-lg bg-black text-white hover:bg-black/80 disabled:opacity-50"
              >
                {topupBusy ? 'Processing...' : 'Add Funds'}
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
