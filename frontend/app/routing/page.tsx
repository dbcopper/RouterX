'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet } from '@/lib/api';

export default function RoutingPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/routing-rules', token)
      .then(setItems)
      .catch((err) => setError(err.message || 'Failed to load'));
  }, []);

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Routing Rules</h1>
            <p className="text-sm text-black/60">Primary/secondary fallback with capability matching.</p>
          </div>
          <Nav />
        </div>
        {error && <p className="text-red-500">{error}</p>}
        <div className="space-y-3">
          {items.map((r) => (
            <div key={r.id} className="card p-4">
              <div className="flex justify-between">
                <strong>{r.capability}</strong>
                <span className="text-xs text-black/60">Tenant: {r.tenant_id}</span>
              </div>
              <p className="text-sm">Primary: {r.primary_provider_id} | Secondary: {r.secondary_provider_id || 'none'}</p>
              <p className="text-xs text-black/60">Model: {r.model}</p>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
