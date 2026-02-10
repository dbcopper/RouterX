'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet } from '@/lib/api';

export default function TenantsPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/tenants', token)
      .then(setItems)
      .catch((err) => setError(err.message || 'Failed to load'));
  }, []);

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Tenants & API Keys</h1>
            <p className="text-sm text-black/60">Manage tenant isolation and quotas.</p>
          </div>
          <Nav />
        </div>
        {error && <p className="text-red-500">{error}</p>}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {items.map((t) => (
            <div key={t.id} className="card p-4">
              <h3 className="font-semibold">{t.name}</h3>
              <p className="text-xs text-black/60">ID: {t.id}</p>
              <p className="text-xs text-black/60">API Keys managed via backend seed or admin tooling.</p>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
