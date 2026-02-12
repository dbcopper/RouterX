'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import StatusBadge from '@/components/StatusBadge';
import { apiGet } from '@/lib/api';
import { Area, Bar, BarChart, ComposedChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis, CartesianGrid, Cell } from 'recharts';

interface DashboardStats {
  total_tenants: number;
  active_tenants: number;
  requests_24h: number;
  errors_24h: number;
  error_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  cost_24h: number;
  tokens_24h: number;
  hourly_series: { hour: string; requests: number; errors: number }[];
}

interface ProviderHealth {
  provider_id: string;
  provider_name: string;
  type: string;
  enabled: boolean;
  health_status: string;
  circuit_open: boolean;
}

interface ModelUsage {
  model: string;
  provider: string;
  requests: number;
  tokens: number;
  cost_usd: number;
}

function StatCard({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <div className="card p-5">
      <p className="text-xs text-black/50 uppercase tracking-wide">{label}</p>
      <p className="text-2xl font-semibold mt-1">{value}</p>
      {sub && <p className="text-xs text-black/40 mt-0.5">{sub}</p>}
    </div>
  );
}

function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse bg-black/5 rounded ${className}`} />;
}

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [health, setHealth] = useState<ProviderHealth[]>([]);
  const [modelUsage, setModelUsage] = useState<ModelUsage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  function load() {
    const token = localStorage.getItem('routerx_token') || '';
    setLoading(true);
    setError('');
    Promise.all([
      apiGet('/admin/stats', token),
      apiGet('/admin/provider-health', token),
      apiGet('/admin/model-usage', token)
    ])
      .then(([s, h, m]) => {
        setStats(s);
        setHealth(Array.isArray(h) ? h : []);
        setModelUsage(Array.isArray(m) ? m : []);
      })
      .catch((e) => setError(e.message || 'Failed to load'))
      .finally(() => setLoading(false));
  }

  useEffect(() => { load(); }, []);

  const hourlyData = (stats?.hourly_series || []).map((b) => ({
    hour: new Date(b.hour).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    requests: b.requests,
    errors: b.errors
  }));

  const modelColors = ['#0f6b4b', '#2563eb', '#f97316', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16', '#ef4444'];

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Dashboard</h1>
            <p className="text-sm text-black/50">Multi-provider routing health at a glance</p>
          </div>
          <Nav />
        </div>

        {error && (
          <div className="card p-4 border-red-200 bg-red-50 flex items-center justify-between">
            <span className="text-sm text-red-600">{error}</span>
            <button onClick={load} className="text-sm text-red-600 underline">Retry</button>
          </div>
        )}

        {/* KPI Cards */}
        {loading ? (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-20" />)}
          </div>
        ) : stats && (
          <>
            <section className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <StatCard label="Tenants" value={stats.total_tenants} sub={`${stats.active_tenants} active`} />
              <StatCard label="Requests (24h)" value={stats.requests_24h.toLocaleString()} />
              <StatCard label="Errors (24h)" value={stats.errors_24h} sub={`${stats.error_rate.toFixed(1)}% error rate`} />
              <StatCard label="Tokens (24h)" value={stats.tokens_24h.toLocaleString()} />
              <StatCard label="Avg Latency" value={`${Math.round(stats.avg_latency_ms)} ms`} />
              <StatCard label="P95 Latency" value={`${Math.round(stats.p95_latency_ms)} ms`} />
              <StatCard label="Cost (24h)" value={`$${stats.cost_24h.toFixed(4)}`} />
              <StatCard label="Providers" value={health.length} sub={`${health.filter(h => h.enabled).length} enabled`} />
            </section>
          </>
        )}

        {/* Provider Health */}
        {!loading && health.length > 0 && (
          <section className="card p-6">
            <h2 className="text-lg font-semibold mb-4">Provider Health</h2>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {health.map((p) => (
                <div key={p.provider_id} className="flex items-center gap-3 p-3 rounded-lg border border-black/5 bg-black/[0.02]">
                  <StatusBadge
                    status={!p.enabled ? 'inactive' : p.circuit_open ? 'fail' : p.health_status === 'ok' ? 'ok' : p.health_status === 'fail' ? 'fail' : 'unknown'}
                  />
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{p.provider_name}</p>
                    <p className="text-xs text-black/40">{p.type}{p.circuit_open ? ' · circuit open' : ''}{!p.enabled ? ' · disabled' : ''}</p>
                  </div>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Hourly Chart */}
        {!loading && hourlyData.length > 0 && (
          <section className="card p-6">
            <h2 className="text-lg font-semibold mb-4">Requests & Errors (Last 24h)</h2>
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <ComposedChart data={hourlyData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="hour" tick={{ fontSize: 11 }} interval={2} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip
                    contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb', fontSize: '12px' }}
                  />
                  <Area type="monotone" dataKey="requests" stroke="#0f6b4b" fill="#a7f3d0" fillOpacity={0.5} name="Requests" />
                  <Line type="monotone" dataKey="errors" stroke="#ef4444" strokeWidth={2} dot={false} name="Errors" />
                </ComposedChart>
              </ResponsiveContainer>
            </div>
          </section>
        )}

        {/* Model Usage */}
        {!loading && (
          <section className="card p-6">
            <h2 className="text-lg font-semibold mb-4">Model Usage</h2>
            {modelUsage.length === 0 ? (
              <p className="text-sm text-black/40">No usage data yet.</p>
            ) : (
              <>
                <div className="h-48 mb-6">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={modelUsage.slice(0, 8)} layout="vertical">
                      <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                      <XAxis type="number" tick={{ fontSize: 11 }} />
                      <YAxis dataKey="model" type="category" tick={{ fontSize: 11 }} width={140} />
                      <Tooltip contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb', fontSize: '12px' }} />
                      <Bar dataKey="tokens" name="Tokens" radius={[0, 4, 4, 0]}>
                        {modelUsage.slice(0, 8).map((_, i) => (
                          <Cell key={i} fill={modelColors[i % modelColors.length]} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
                <div className="border rounded-lg overflow-hidden text-sm">
                  <div className="grid grid-cols-12 gap-2 px-4 py-2.5 bg-black/[0.03] text-xs font-medium text-black/60 uppercase tracking-wide">
                    <div className="col-span-4">Model</div>
                    <div className="col-span-2">Provider</div>
                    <div className="col-span-2 text-right">Requests</div>
                    <div className="col-span-2 text-right">Tokens</div>
                    <div className="col-span-2 text-right">Cost</div>
                  </div>
                  {modelUsage.map((m, i) => (
                    <div key={`${m.model}-${m.provider}-${i}`} className="grid grid-cols-12 gap-2 px-4 py-2.5 border-t border-black/5 hover:bg-black/[0.02]">
                      <div className="col-span-4 font-medium">{m.model}</div>
                      <div className="col-span-2 text-black/60">{m.provider}</div>
                      <div className="col-span-2 text-right">{m.requests.toLocaleString()}</div>
                      <div className="col-span-2 text-right">{m.tokens.toLocaleString()}</div>
                      <div className="col-span-2 text-right">${Number(m.cost_usd || 0).toFixed(4)}</div>
                    </div>
                  ))}
                </div>
              </>
            )}
          </section>
        )}
      </div>
    </main>
  );
}
