'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import Nav from '@/components/Nav';
import StatusBadge from '@/components/StatusBadge';
import { apiGet, apiPost } from '@/lib/api';

interface Tenant {
  id: string;
  name: string;
  balance_usd: number;
  created_at: string;
  last_active: string | null;
  suspended: boolean;
  total_topup_usd: number;
  total_spent_usd: number;
}

export default function TenantsPage() {
  const [items, setItems] = useState<Tenant[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [search, setSearch] = useState('');
  const [balanceModal, setBalanceModal] = useState<Tenant | null>(null);
  const [newBalance, setNewBalance] = useState('');
  const [balanceDesc, setBalanceDesc] = useState('');
  const [saving, setSaving] = useState(false);

  function token() {
    return typeof window !== 'undefined' ? localStorage.getItem('routerx_token') || '' : '';
  }

  async function refresh() {
    try {
      const list = await apiGet('/admin/tenants', token());
      setItems(Array.isArray(list) ? list : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load');
    }
  }

  useEffect(() => { refresh(); }, []);

  function isActive(t: Tenant) {
    if (!t.last_active) return false;
    return Date.now() - new Date(t.last_active).getTime() < 24 * 60 * 60 * 1000;
  }

  async function saveBalance() {
    if (!balanceModal) return;
    setSaving(true);
    setError('');
    try {
      await apiPost(`/admin/tenants/${balanceModal.id}/balance`, {
        balance_usd: parseFloat(newBalance),
        description: balanceDesc || undefined
      }, token());
      setBalanceModal(null);
      setStatus('Balance updated');
      refresh();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  async function toggleSuspend(t: Tenant) {
    setError('');
    try {
      const action = t.suspended ? 'unsuspend' : 'suspend';
      await apiPost(`/admin/tenants/${t.id}/${action}`, {}, token());
      setStatus(`${t.name} ${t.suspended ? 'unsuspended' : 'suspended'}`);
      refresh();
    } catch (err: any) {
      setError(err.message);
    }
  }

  const filtered = items.filter((t) =>
    !search || t.name.toLowerCase().includes(search.toLowerCase()) || t.id.includes(search)
  );

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Tenants</h1>
            <p className="text-sm text-black/50">Manage tenant isolation, billing, and suspension</p>
          </div>
          <Nav />
        </div>

        {error && (
          <div className="card p-3 border-red-200 bg-red-50 text-sm text-red-600 flex items-center justify-between">
            <span>{error}</span>
            <button onClick={() => setError('')} className="text-red-400">✕</button>
          </div>
        )}
        {status && (
          <div className="card p-3 border-green-200 bg-green-50 text-sm text-green-700 flex items-center justify-between">
            <span>{status}</span>
            <button onClick={() => setStatus('')} className="text-green-400">✕</button>
          </div>
        )}

        <div className="card p-4">
          <div className="flex items-center justify-between mb-4">
            <input
              type="text"
              placeholder="Search by name or ID..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="px-3 py-1.5 text-sm border border-black/10 rounded-lg bg-white w-64"
            />
            <button onClick={refresh} className="text-sm px-3 py-1.5 rounded-lg border border-black/10 hover:bg-black/5">
              Refresh
            </button>
          </div>

          <div className="border rounded-lg overflow-hidden text-sm">
            <table className="w-full">
              <thead>
                <tr className="bg-black/[0.03] text-xs font-medium text-black/60 uppercase tracking-wide">
                  <th className="px-4 py-3 text-left">Status</th>
                  <th className="px-4 py-3 text-left">Tenant</th>
                  <th className="px-4 py-3 text-right">Balance</th>
                  <th className="px-4 py-3 text-right">Total Spent</th>
                  <th className="px-4 py-3 text-right">Total Topup</th>
                  <th className="px-4 py-3 text-left">Last Active</th>
                  <th className="px-4 py-3 text-center w-48">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 ? (
                  <tr><td colSpan={7} className="px-4 py-8 text-center text-black/40">No tenants found</td></tr>
                ) : filtered.map((t) => (
                  <tr key={t.id} className="border-t border-black/5 hover:bg-black/[0.02]">
                    <td className="px-4 py-3">
                      {t.suspended ? (
                        <span className="inline-block px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-600">Suspended</span>
                      ) : (
                        <StatusBadge status={isActive(t) ? 'active' : 'inactive'} label={isActive(t) ? 'Active' : 'Inactive'} />
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <Link href={`/tenants/${t.id}`} className="font-medium hover:underline">{t.name}</Link>
                      <div className="text-xs text-black/40 font-mono">{t.id}</div>
                    </td>
                    <td className="px-4 py-3 text-right font-mono">
                      <span className={Number(t.balance_usd) <= 0 ? 'text-red-500' : ''}>
                        ${Number(t.balance_usd || 0).toFixed(2)}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right font-mono text-black/60">${Number(t.total_spent_usd || 0).toFixed(4)}</td>
                    <td className="px-4 py-3 text-right font-mono text-black/60">${Number(t.total_topup_usd || 0).toFixed(2)}</td>
                    <td className="px-4 py-3 text-black/60 whitespace-nowrap text-xs">
                      {t.last_active ? new Date(t.last_active).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : 'Never'}
                    </td>
                    <td className="px-4 py-3 text-center space-x-2">
                      <button
                        onClick={() => { setBalanceModal(t); setNewBalance(String(t.balance_usd || 0)); setBalanceDesc(''); }}
                        className="text-xs px-3 py-1 rounded-lg border border-black/10 hover:bg-black/5"
                      >
                        Adjust
                      </button>
                      <button
                        onClick={() => toggleSuspend(t)}
                        className={`text-xs px-3 py-1 rounded-lg border ${
                          t.suspended
                            ? 'border-green-300 text-green-600 hover:bg-green-50'
                            : 'border-red-200 text-red-500 hover:bg-red-50'
                        }`}
                      >
                        {t.suspended ? 'Unsuspend' : 'Suspend'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Balance Modal */}
      {balanceModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 max-w-sm w-full mx-4">
            <h3 className="font-semibold text-lg mb-1">Adjust Balance</h3>
            <p className="text-sm text-black/50 mb-4">
              Tenant: <span className="font-medium text-black">{balanceModal.name}</span>
            </p>
            <p className="text-xs text-black/40 mb-2">
              Current balance: ${Number(balanceModal.balance_usd || 0).toFixed(2)}
            </p>
            <label className="text-sm font-medium">New Balance (USD)</label>
            <input
              type="number"
              step="0.01"
              value={newBalance}
              onChange={(e) => setNewBalance(e.target.value)}
              className="w-full mt-1 px-3 py-2 border border-black/10 rounded-lg text-sm"
              autoFocus
            />
            <label className="text-sm font-medium mt-3 block">Description (optional)</label>
            <input
              type="text"
              value={balanceDesc}
              onChange={(e) => setBalanceDesc(e.target.value)}
              placeholder="e.g. Manual topup, Credit adjustment"
              className="w-full mt-1 px-3 py-2 border border-black/10 rounded-lg text-sm"
            />
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setBalanceModal(null)}
                className="px-4 py-2 text-sm rounded-lg border border-black/10 hover:bg-black/5"
              >
                Cancel
              </button>
              <button
                onClick={saveBalance}
                disabled={saving}
                className="px-4 py-2 text-sm rounded-lg bg-black text-white hover:bg-black/80 disabled:opacity-50"
              >
                {saving ? 'Saving...' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
