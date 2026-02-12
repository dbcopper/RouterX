'use client';

import { useEffect, useState, useCallback } from 'react';
import Nav from '@/components/Nav';
import Pagination from '@/components/Pagination';
import { apiGet, apiDelete } from '@/lib/api';

interface RequestLog {
  id: number;
  tenant_id: string;
  provider: string;
  model: string;
  latency_ms: number;
  ttft_ms: number;
  tokens: number;
  cost_usd: number;
  prompt_hash: string;
  fallback_used: boolean;
  status_code: number;
  error_code: string;
  created_at: string;
}

interface PaginatedResult {
  data: RequestLog[];
  total: number;
  page: number;
  page_size: number;
}

type SortField = 'created_at' | 'latency_ms' | 'tokens' | 'cost_usd' | 'model' | 'provider';

export default function RequestsPage() {
  const [result, setResult] = useState<PaginatedResult>({ data: [], total: 0, page: 1, page_size: 50 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [page, setPage] = useState(1);
  const [tenantFilter, setTenantFilter] = useState('');
  const [providerFilter, setProviderFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [sortBy, setSortBy] = useState<SortField>('created_at');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');

  const [debouncedTenant, setDebouncedTenant] = useState('');
  const [debouncedModel, setDebouncedModel] = useState('');

  useEffect(() => {
    const t = setTimeout(() => setDebouncedTenant(tenantFilter), 300);
    return () => clearTimeout(t);
  }, [tenantFilter]);

  useEffect(() => {
    const t = setTimeout(() => setDebouncedModel(modelFilter), 300);
    return () => clearTimeout(t);
  }, [modelFilter]);

  const fetchData = useCallback(() => {
    const token = localStorage.getItem('routerx_token') || '';
    setLoading(true);
    setError('');
    const params = new URLSearchParams({ page: String(page), page_size: '50', sort_by: sortBy, sort_dir: sortDir });
    if (debouncedTenant) params.set('tenant_id', debouncedTenant);
    if (providerFilter) params.set('provider', providerFilter);
    if (debouncedModel) params.set('model', debouncedModel);
    if (statusFilter) params.set('status_code', statusFilter);
    apiGet(`/admin/requests?${params.toString()}`, token)
      .then((r) => setResult({ ...r, data: r.data || [] }))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [page, debouncedTenant, providerFilter, debouncedModel, statusFilter, sortBy, sortDir]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useEffect(() => { setPage(1); }, [debouncedTenant, providerFilter, debouncedModel, statusFilter]);

  function handleSort(field: SortField) {
    if (sortBy === field) {
      setSortDir(sortDir === 'desc' ? 'asc' : 'desc');
    } else {
      setSortBy(field);
      setSortDir('desc');
    }
  }

  function handleDelete(id: number) {
    const token = localStorage.getItem('routerx_token') || '';
    apiDelete(`/admin/requests/${id}`, token).then(() => fetchData());
  }

  function SortIcon({ field }: { field: SortField }) {
    if (sortBy !== field) return <span className="text-black/20 ml-1">↕</span>;
    return <span className="ml-1">{sortDir === 'desc' ? '↓' : '↑'}</span>;
  }

  const statusColor = (code: number) => {
    if (code >= 200 && code < 300) return 'text-emerald-600 bg-emerald-50';
    if (code >= 400 && code < 500) return 'text-amber-600 bg-amber-50';
    return 'text-red-600 bg-red-50';
  };

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Request Logs</h1>
            <p className="text-sm text-black/50">Audit trail for all API requests</p>
          </div>
          <Nav />
        </div>

        {/* Filters */}
        <div className="card p-4">
          <div className="flex flex-wrap items-center gap-3">
            <input
              type="text"
              placeholder="Filter by tenant ID..."
              value={tenantFilter}
              onChange={(e) => setTenantFilter(e.target.value)}
              className="px-3 py-1.5 text-sm border border-black/10 rounded-lg bg-white w-44"
            />
            <input
              type="text"
              placeholder="Filter by provider..."
              value={providerFilter}
              onChange={(e) => setProviderFilter(e.target.value)}
              className="px-3 py-1.5 text-sm border border-black/10 rounded-lg bg-white w-40"
            />
            <input
              type="text"
              placeholder="Filter by model..."
              value={modelFilter}
              onChange={(e) => setModelFilter(e.target.value)}
              className="px-3 py-1.5 text-sm border border-black/10 rounded-lg bg-white w-40"
            />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="px-3 py-1.5 text-sm border border-black/10 rounded-lg bg-white"
            >
              <option value="">All statuses</option>
              <option value="200">200 OK</option>
              <option value="400">400 Bad Request</option>
              <option value="402">402 Payment Required</option>
              <option value="429">429 Rate Limited</option>
              <option value="502">502 Bad Gateway</option>
            </select>
            <button
              onClick={fetchData}
              className="px-3 py-1.5 text-sm rounded-lg border border-black/10 hover:bg-black/5 ml-auto"
            >
              Refresh
            </button>
          </div>
        </div>

        {error && (
          <div className="card p-3 border-red-200 bg-red-50 text-sm text-red-600">{error}</div>
        )}

        {/* Table */}
        <div className="card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-black/[0.03] text-xs font-medium text-black/60 uppercase tracking-wide">
                  <th className="px-4 py-3 text-left cursor-pointer select-none" onClick={() => handleSort('created_at')}>
                    Time <SortIcon field="created_at" />
                  </th>
                  <th className="px-4 py-3 text-left">Tenant</th>
                  <th className="px-4 py-3 text-left cursor-pointer select-none" onClick={() => handleSort('provider')}>
                    Provider <SortIcon field="provider" />
                  </th>
                  <th className="px-4 py-3 text-left cursor-pointer select-none" onClick={() => handleSort('model')}>
                    Model <SortIcon field="model" />
                  </th>
                  <th className="px-4 py-3 text-right cursor-pointer select-none" onClick={() => handleSort('latency_ms')}>
                    Latency <SortIcon field="latency_ms" />
                  </th>
                  <th className="px-4 py-3 text-right">TTFT</th>
                  <th className="px-4 py-3 text-right cursor-pointer select-none" onClick={() => handleSort('tokens')}>
                    Tokens <SortIcon field="tokens" />
                  </th>
                  <th className="px-4 py-3 text-right cursor-pointer select-none" onClick={() => handleSort('cost_usd')}>
                    Cost <SortIcon field="cost_usd" />
                  </th>
                  <th className="px-4 py-3 text-center">Status</th>
                  <th className="px-4 py-3 text-center">Fallback</th>
                  <th className="px-4 py-3 text-center w-16"></th>
                </tr>
              </thead>
              <tbody>
                {loading && !result.data.length ? (
                  <tr><td colSpan={11} className="px-4 py-8 text-center text-black/40">Loading...</td></tr>
                ) : result.data.length === 0 ? (
                  <tr><td colSpan={11} className="px-4 py-8 text-center text-black/40">No requests found</td></tr>
                ) : (
                  result.data.map((r) => (
                    <tr key={r.id} className="border-t border-black/5 hover:bg-black/[0.02]">
                      <td className="px-4 py-2.5 whitespace-nowrap text-black/60">
                        {new Date(r.created_at).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                      </td>
                      <td className="px-4 py-2.5 font-mono text-xs text-black/50 max-w-[100px] truncate">{r.tenant_id}</td>
                      <td className="px-4 py-2.5">{r.provider}</td>
                      <td className="px-4 py-2.5 font-medium">{r.model}</td>
                      <td className="px-4 py-2.5 text-right font-mono">{r.latency_ms} ms</td>
                      <td className="px-4 py-2.5 text-right font-mono text-black/50">{r.ttft_ms} ms</td>
                      <td className="px-4 py-2.5 text-right">{r.tokens.toLocaleString()}</td>
                      <td className="px-4 py-2.5 text-right font-mono">${Number(r.cost_usd).toFixed(4)}</td>
                      <td className="px-4 py-2.5 text-center">
                        <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${statusColor(r.status_code)}`}>
                          {r.status_code}
                        </span>
                      </td>
                      <td className="px-4 py-2.5 text-center">
                        {r.fallback_used && <span className="text-xs text-amber-600">Yes</span>}
                      </td>
                      <td className="px-4 py-2.5 text-center">
                        <button
                          onClick={() => handleDelete(r.id)}
                          className="text-xs text-red-400 hover:text-red-600"
                          title="Delete"
                        >
                          ✕
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          <div className="px-4 pb-3">
            <Pagination page={page} pageSize={result.page_size} total={result.total} onChange={setPage} />
          </div>
        </div>
      </div>
    </main>
  );
}
