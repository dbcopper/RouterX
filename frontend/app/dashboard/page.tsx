'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
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
  total_requests_all_time: number;
  total_tokens_all_time: number;
  total_cost_all_time: number;
  total_revenue_all_time: number;
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

interface TenantRow {
  id: string;
  name: string;
  balance_usd: number;
  suspended: boolean;
  total_topup_usd: number;
  total_spent_usd: number;
  last_active: string | null;
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
  const [tenants, setTenants] = useState<TenantRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  function load() {
    const token = localStorage.getItem('routerx_token') || '';
    setLoading(true);
    setError('');
    Promise.all([
      apiGet('/admin/stats', token),
      apiGet('/admin/provider-health', token),
      apiGet('/admin/model-usage', token),
      apiGet('/admin/tenants', token)
    ])
      .then(([s, h, m, t]) => {
        setStats(s);
        setHealth(Array.isArray(h) ? h : []);
        setModelUsage(Array.isArray(m) ? m : []);
        setTenants(Array.isArray(t) ? t : []);
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

        {/* KPI Cards — All-time */}
        {loading ? (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-20" />)}
          </div>
        ) : stats && (
          <>
            <section>
              <h2 className="text-sm font-semibold text-black/50 uppercase tracking-wide mb-3">All-Time</h2>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <StatCard label="Total Requests" value={stats.total_requests_all_time.toLocaleString()} />
                <StatCard label="Total Tokens" value={stats.total_tokens_all_time.toLocaleString()} />
                <StatCard label="Total Cost" value={`$${stats.total_cost_all_time.toFixed(4)}`} />
                <StatCard label="Total Revenue" value={`$${stats.total_revenue_all_time.toFixed(2)}`} sub="Sum of all topups" />
              </div>
            </section>

            <section>
              <h2 className="text-sm font-semibold text-black/50 uppercase tracking-wide mb-3">Last 24 Hours</h2>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <StatCard label="Tenants" value={stats.total_tenants} sub={`${stats.active_tenants} active`} />
                <StatCard label="Requests" value={stats.requests_24h.toLocaleString()} sub={`${stats.errors_24h} errors (${stats.error_rate.toFixed(1)}%)`} />
                <StatCard label="Tokens" value={stats.tokens_24h.toLocaleString()} />
                <StatCard label="Cost" value={`$${stats.cost_24h.toFixed(4)}`} sub={`P95: ${Math.round(stats.p95_latency_ms)}ms`} />
              </div>
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

        {/* Tenant Overview */}
        {!loading && tenants.length > 0 && (
          <section className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">Tenant Overview</h2>
              <Link href="/tenants" className="text-sm text-black/50 hover:text-black underline">View All</Link>
            </div>
            <div className="border rounded-lg overflow-hidden text-sm">
              <table className="w-full">
                <thead>
                  <tr className="bg-black/[0.03] text-xs font-medium text-black/60 uppercase tracking-wide">
                    <th className="px-4 py-2.5 text-left">Tenant</th>
                    <th className="px-4 py-2.5 text-center">Status</th>
                    <th className="px-4 py-2.5 text-right">Balance</th>
                    <th className="px-4 py-2.5 text-right">Total Spent</th>
                    <th className="px-4 py-2.5 text-right">Total Topup</th>
                    <th className="px-4 py-2.5 text-left">Last Active</th>
                  </tr>
                </thead>
                <tbody>
                  {tenants.slice(0, 10).map((t) => {
                    const active = t.last_active && (Date.now() - new Date(t.last_active).getTime() < 24 * 60 * 60 * 1000);
                    return (
                      <tr key={t.id} className="border-t border-black/5 hover:bg-black/[0.02]">
                        <td className="px-4 py-2.5">
                          <Link href={`/tenants/${t.id}`} className="font-medium hover:underline">{t.name}</Link>
                        </td>
                        <td className="px-4 py-2.5 text-center">
                          {t.suspended ? (
                            <span className="inline-block px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-600">Suspended</span>
                          ) : (
                            <StatusBadge status={active ? 'active' : 'inactive'} label={active ? 'Active' : 'Inactive'} />
                          )}
                        </td>
                        <td className="px-4 py-2.5 text-right font-mono">
                          <span className={Number(t.balance_usd) <= 0 ? 'text-red-500' : ''}>${Number(t.balance_usd || 0).toFixed(2)}</span>
                        </td>
                        <td className="px-4 py-2.5 text-right font-mono text-black/60">${Number(t.total_spent_usd || 0).toFixed(4)}</td>
                        <td className="px-4 py-2.5 text-right font-mono text-black/60">${Number(t.total_topup_usd || 0).toFixed(2)}</td>
                        <td className="px-4 py-2.5 text-black/60 whitespace-nowrap text-xs">
                          {t.last_active ? new Date(t.last_active).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : 'Never'}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
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
