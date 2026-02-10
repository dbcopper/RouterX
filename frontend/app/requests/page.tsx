'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet } from '@/lib/api';

export default function RequestsPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/requests', token)
      .then(setItems)
      .catch((err) => setError(err.message || 'Failed to load'));
  }, []);

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Requests & Audit</h1>
            <p className="text-sm text-black/60">Metadata-only logs for compliance and cost tracking.</p>
          </div>
          <Nav />
        </div>
        {error && <p className="text-red-500">{error}</p>}
        <div className="card p-4">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b">
                <th className="py-2">Tenant</th>
                <th>Provider</th>
                <th>Model</th>
                <th>Latency (ms)</th>
                <th>Tokens</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {items.map((l, idx) => (
                <tr key={idx} className="border-b last:border-0">
                  <td className="py-2">{l.tenant_id}</td>
                  <td>{l.provider}</td>
                  <td>{l.model}</td>
                  <td>{l.latency_ms}</td>
                  <td>{l.tokens}</td>
                  <td>{l.status_code}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </main>
  );
}
