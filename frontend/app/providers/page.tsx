'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet } from '@/lib/api';

export default function ProvidersPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/providers', token)
      .then(setItems)
      .catch((err) => setError(err.message || 'Failed to load'));
  }, []);

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Providers</h1>
            <p className="text-sm text-black/60">Native and generic OpenAI-compatible adapters.</p>
          </div>
          <Nav />
        </div>
        {error && <p className="text-red-500">{error}</p>}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {items.map((p) => (
            <div key={p.id} className="card p-4">
              <h3 className="font-semibold">{p.name}</h3>
              <p className="text-xs text-black/60">Type: {p.type}</p>
              <p className="text-xs text-black/60">Text: {String(p.supports_text)} | Vision: {String(p.supports_vision)}</p>
              <p className="text-xs text-black/60">Enabled: {String(p.enabled)}</p>
              {p.base_url && <p className="text-xs text-black/60">Base URL: {p.base_url}</p>}
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
