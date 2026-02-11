'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet, apiPut } from '@/lib/api';

export default function TenantsPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');

  async function refresh() {
    const token = localStorage.getItem('routerx_token') || '';
    try {
      const list = await apiGet('/admin/tenants', token);
      setItems(Array.isArray(list) ? list : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load');
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  function updateField(id: string, value: string) {
    setItems((prev) => prev.map((t) => (t.id === id ? { ...t, balance_usd: value } : t)));
  }

  async function saveBalance(t: any) {
    setStatus('');
    setError('');
    try {
      const token = localStorage.getItem('routerx_token') || '';
      await apiPut(`/admin/tenants/${t.id}/balance`, { balance_usd: Number(t.balance_usd) }, token);
      setStatus(`Balance updated for ${t.name}`);
    } catch (err: any) {
      setError(err.message || 'Failed to update balance');
    }
  }

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Tenants & Balances</h1>
            <p className="text-sm text-black/60">Manage tenant isolation and billing balance.</p>
          </div>
          <Nav />
        </div>
        {error && <p className="text-red-500">{error}</p>}
        {status && <p className="text-green-600">{status}</p>}
        <div className="card p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="font-semibold">Tenants</h2>
            <button className="text-sm underline" onClick={refresh}>Refresh</button>
          </div>
          <div className="border rounded-lg overflow-hidden text-sm">
            <div className="grid grid-cols-12 gap-2 px-3 py-2 bg-black/5 text-xs font-semibold">
              <div className="col-span-2">Created</div>
              <div className="col-span-2">Last Active</div>
              <div className="col-span-3">Tenant</div>
              <div className="col-span-2">ID</div>
              <div className="col-span-2">Balance (USD)</div>
              <div className="col-span-1">Action</div>
            </div>
            {items.map((t) => (
              <div key={t.id} className="grid grid-cols-12 gap-2 px-3 py-2 border-t items-center">
                <div className="col-span-2 text-xs text-black/70">{t.created_at ? new Date(t.created_at).toLocaleString() : '-'}</div>
                <div className="col-span-2 text-xs text-black/70">{t.last_active ? new Date(t.last_active).toLocaleString() : '-'}</div>
                <div className="col-span-3 font-semibold">{t.name}</div>
                <div className="col-span-2 text-xs text-black/70">{t.id}</div>
                <div className="col-span-2">
                  <input className="w-full border border-black/10 rounded-lg px-2 py-1" value={t.balance_usd ?? 0} onChange={(e) => updateField(t.id, e.target.value)} />
                </div>
                <div className="col-span-1">
                  <button className="px-2 py-1 rounded-lg bg-ink text-white text-xs" onClick={() => saveBalance(t)}>Save</button>
                </div>
              </div>
            ))}
            {!items.length && (
              <div className="px-3 py-2 text-sm text-black/50">No tenants found.</div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
