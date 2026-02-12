'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Nav from '@/components/Nav';
import StatusBadge from '@/components/StatusBadge';
import { apiGet, apiPost, apiPut } from '@/lib/api';

interface TenantDetail {
  id: string;
  name: string;
  balance_usd: number;
  created_at: string;
  last_active: string | null;
  suspended: boolean;
  total_topup_usd: number;
  total_spent_usd: number;
  rate_limit_rpm: number;
  spend_limit_usd: number;
}

interface Transaction {
  id: number;
  tenant_id: string;
  type: string;
  amount_usd: number;
  balance_after: number;
  description: string;
  created_at: string;
}

export default function TenantDetailPage() {
  const params = useParams();
  const router = useRouter();
  const tenantId = params.id as string;

  const [tenant, setTenant] = useState<TenantDetail | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');

  // Balance adjustment
  const [showAdjust, setShowAdjust] = useState(false);
  const [newBalance, setNewBalance] = useState('');
  const [adjustDesc, setAdjustDesc] = useState('');
  const [saving, setSaving] = useState(false);

  // Limits
  const [editRPM, setEditRPM] = useState('');
  const [editSpendLimit, setEditSpendLimit] = useState('');
  const [showLimits, setShowLimits] = useState(false);
  const [savingLimits, setSavingLimits] = useState(false);

  function token() {
    return typeof window !== 'undefined' ? localStorage.getItem('routerx_token') || '' : '';
  }

  async function load() {
    setLoading(true);
    setError('');
    try {
      const [t, txs] = await Promise.all([
        apiGet(`/admin/tenants/${tenantId}`, token()),
        apiGet(`/admin/tenants/${tenantId}/transactions`, token())
      ]);
      setTenant(t);
      setTransactions(Array.isArray(txs) ? txs : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, [tenantId]);

  async function toggleSuspend() {
    if (!tenant) return;
    setError('');
    try {
      const action = tenant.suspended ? 'unsuspend' : 'suspend';
      await apiPost(`/admin/tenants/${tenantId}/${action}`, {}, token());
      setStatus(`Tenant ${tenant.suspended ? 'unsuspended' : 'suspended'}`);
      load();
    } catch (err: any) {
      setError(err.message);
    }
  }

  async function saveBalance() {
    setSaving(true);
    setError('');
    try {
      await apiPost(`/admin/tenants/${tenantId}/balance`, {
        balance_usd: parseFloat(newBalance),
        description: adjustDesc || undefined
      }, token());
      setShowAdjust(false);
      setStatus('Balance updated');
      load();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  async function saveLimits() {
    setSavingLimits(true);
    setError('');
    try {
      await apiPut(`/admin/tenants/${tenantId}/limits`, {
        rate_limit_rpm: parseInt(editRPM) || 60,
        spend_limit_usd: parseFloat(editSpendLimit) || 0
      }, token());
      setShowLimits(false);
      setStatus('Limits updated');
      load();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSavingLimits(false);
    }
  }

  function txColor(type: string) {
    if (type === 'topup') return 'text-green-600';
    if (type === 'charge') return 'text-red-500';
    return 'text-blue-600';
  }

  function txBadge(type: string) {
    const colors: Record<string, string> = {
      topup: 'bg-green-50 text-green-700',
      charge: 'bg-red-50 text-red-600',
      adjustment: 'bg-blue-50 text-blue-700'
    };
    return colors[type] || 'bg-gray-50 text-gray-700';
  }

  if (loading) {
    return (
      <main className="min-h-screen p-8">
        <div className="max-w-5xl mx-auto">
          <div className="flex items-center justify-between mb-6">
            <div className="animate-pulse bg-black/5 rounded h-8 w-48" />
            <Nav />
          </div>
          <div className="space-y-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="animate-pulse bg-black/5 rounded h-24" />
            ))}
          </div>
        </div>
      </main>
    );
  }

  if (!tenant) {
    return (
      <main className="min-h-screen p-8">
        <div className="max-w-5xl mx-auto">
          <div className="flex items-center justify-between mb-6">
            <h1 className="text-3xl font-semibold">Tenant Not Found</h1>
            <Nav />
          </div>
          <p className="text-black/50">The tenant with ID "{tenantId}" could not be found.</p>
        </div>
      </main>
    );
  }

  const isActive = tenant.last_active && (Date.now() - new Date(tenant.last_active).getTime() < 24 * 60 * 60 * 1000);

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <button onClick={() => router.push('/tenants')} className="text-sm text-black/40 hover:text-black mb-1 block">&larr; Back to Tenants</button>
            <h1 className="text-3xl font-semibold">{tenant.name}</h1>
            <p className="text-sm text-black/50 font-mono">{tenant.id}</p>
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

        {/* Profile Card */}
        <div className="card p-6">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
            <div>
              <p className="text-xs text-black/50 uppercase tracking-wide">Status</p>
              <div className="mt-1">
                {tenant.suspended ? (
                  <span className="inline-block px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-600">Suspended</span>
                ) : (
                  <StatusBadge status={isActive ? 'active' : 'inactive'} label={isActive ? 'Active' : 'Inactive'} />
                )}
              </div>
            </div>
            <div>
              <p className="text-xs text-black/50 uppercase tracking-wide">Balance</p>
              <p className={`text-2xl font-semibold mt-1 ${Number(tenant.balance_usd) <= 0 ? 'text-red-500' : ''}`}>
                ${Number(tenant.balance_usd || 0).toFixed(2)}
              </p>
            </div>
            <div>
              <p className="text-xs text-black/50 uppercase tracking-wide">Total Spent</p>
              <p className="text-2xl font-semibold mt-1">${Number(tenant.total_spent_usd || 0).toFixed(4)}</p>
            </div>
            <div>
              <p className="text-xs text-black/50 uppercase tracking-wide">Total Topup</p>
              <p className="text-2xl font-semibold mt-1">${Number(tenant.total_topup_usd || 0).toFixed(2)}</p>
            </div>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-6 mt-4 pt-4 border-t border-black/5">
            <div>
              <p className="text-xs text-black/50 uppercase tracking-wide">Rate Limit</p>
              <p className="text-lg font-semibold mt-1">{tenant.rate_limit_rpm} RPM</p>
            </div>
            <div>
              <p className="text-xs text-black/50 uppercase tracking-wide">Spend Limit</p>
              <p className="text-lg font-semibold mt-1">{tenant.spend_limit_usd > 0 ? `$${Number(tenant.spend_limit_usd).toFixed(2)}` : 'None'}</p>
            </div>
          </div>

          <div className="flex items-center gap-3 mt-6 pt-4 border-t border-black/5">
            <button
              onClick={() => { setShowAdjust(true); setNewBalance(String(tenant.balance_usd || 0)); setAdjustDesc(''); }}
              className="text-sm px-4 py-2 rounded-lg bg-black text-white hover:bg-black/80"
            >
              Adjust Balance
            </button>
            <button
              onClick={() => { setShowLimits(true); setEditRPM(String(tenant.rate_limit_rpm || 60)); setEditSpendLimit(String(tenant.spend_limit_usd || 0)); }}
              className="text-sm px-4 py-2 rounded-lg border border-black/10 hover:bg-black/5"
            >
              Configure Limits
            </button>
            <button
              onClick={toggleSuspend}
              className={`text-sm px-4 py-2 rounded-lg border ${
                tenant.suspended
                  ? 'border-green-300 text-green-700 hover:bg-green-50'
                  : 'border-red-200 text-red-600 hover:bg-red-50'
              }`}
            >
              {tenant.suspended ? 'Unsuspend' : 'Suspend'}
            </button>
            <div className="ml-auto text-xs text-black/40">
              Created: {new Date(tenant.created_at).toLocaleDateString()}
              {tenant.last_active && ` · Last active: ${new Date(tenant.last_active).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}`}
            </div>
          </div>
        </div>

        {/* Transaction History */}
        <div className="card p-6">
          <h2 className="text-lg font-semibold mb-4">Transaction History</h2>
          {transactions.length === 0 ? (
            <p className="text-sm text-black/40">No transactions yet.</p>
          ) : (
            <div className="border rounded-lg overflow-hidden text-sm">
              <table className="w-full">
                <thead>
                  <tr className="bg-black/[0.03] text-xs font-medium text-black/60 uppercase tracking-wide">
                    <th className="px-4 py-2.5 text-left">Date</th>
                    <th className="px-4 py-2.5 text-left">Type</th>
                    <th className="px-4 py-2.5 text-right">Amount</th>
                    <th className="px-4 py-2.5 text-right">Balance After</th>
                    <th className="px-4 py-2.5 text-left">Description</th>
                  </tr>
                </thead>
                <tbody>
                  {transactions.map((tx) => (
                    <tr key={tx.id} className="border-t border-black/5 hover:bg-black/[0.02]">
                      <td className="px-4 py-2.5 text-black/60 whitespace-nowrap text-xs">
                        {new Date(tx.created_at).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                      </td>
                      <td className="px-4 py-2.5">
                        <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${txBadge(tx.type)}`}>
                          {tx.type}
                        </span>
                      </td>
                      <td className={`px-4 py-2.5 text-right font-mono ${txColor(tx.type)}`}>
                        {tx.amount_usd >= 0 ? '+' : ''}{Number(tx.amount_usd).toFixed(4)}
                      </td>
                      <td className="px-4 py-2.5 text-right font-mono text-black/60">
                        ${Number(tx.balance_after).toFixed(4)}
                      </td>
                      <td className="px-4 py-2.5 text-black/50 text-xs max-w-xs truncate">
                        {tx.description || '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* Limits Modal */}
      {showLimits && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 max-w-sm w-full mx-4">
            <h3 className="font-semibold text-lg mb-4">Configure Limits</h3>
            <label className="text-sm font-medium">Rate Limit (requests/min)</label>
            <input
              type="number"
              value={editRPM}
              onChange={(e) => setEditRPM(e.target.value)}
              className="w-full mt-1 px-3 py-2 border border-black/10 rounded-lg text-sm"
            />
            <label className="text-sm font-medium mt-3 block">Spend Limit (USD, 0 = unlimited)</label>
            <input
              type="number"
              step="0.01"
              value={editSpendLimit}
              onChange={(e) => setEditSpendLimit(e.target.value)}
              className="w-full mt-1 px-3 py-2 border border-black/10 rounded-lg text-sm"
            />
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowLimits(false)} className="px-4 py-2 text-sm rounded-lg border border-black/10 hover:bg-black/5">Cancel</button>
              <button onClick={saveLimits} disabled={savingLimits} className="px-4 py-2 text-sm rounded-lg bg-black text-white hover:bg-black/80 disabled:opacity-50">
                {savingLimits ? 'Saving...' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Adjust Balance Modal */}
      {showAdjust && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 max-w-sm w-full mx-4">
            <h3 className="font-semibold text-lg mb-1">Adjust Balance</h3>
            <p className="text-xs text-black/40 mb-4">
              Current balance: ${Number(tenant.balance_usd || 0).toFixed(2)}
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
              value={adjustDesc}
              onChange={(e) => setAdjustDesc(e.target.value)}
              placeholder="e.g. Manual topup"
              className="w-full mt-1 px-3 py-2 border border-black/10 rounded-lg text-sm"
            />
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowAdjust(false)}
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
