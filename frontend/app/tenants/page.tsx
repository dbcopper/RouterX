'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet, apiPut } from '@/lib/api';

export default function TenantsPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/tenants', token)
      .then((list) => setItems(Array.isArray(list) ? list : []))
      .catch((err) => setError(err.message || 'Failed to load'));
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
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {items.map((t) => (
            <div key={t.id} className="card p-4">
              <h3 className="font-semibold">{t.name}</h3>
              <p className="text-xs text-black/60">ID: {t.id}</p>
              <label className="block text-sm mt-3">
                Balance (USD)
                <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" value={t.balance_usd ?? 0} onChange={(e) => updateField(t.id, e.target.value)} />
              </label>
              <button className="mt-3 px-3 py-2 rounded-lg bg-ink text-white" onClick={() => saveBalance(t)}>Save Balance</button>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
