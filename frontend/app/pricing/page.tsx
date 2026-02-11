'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet, apiPost } from '@/lib/api';

export default function PricingPage() {
  const [items, setItems] = useState<any[]>([]);
  const [model, setModel] = useState('');
  const [price, setPrice] = useState('');
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/model-pricing', token)
      .then((list) => setItems(Array.isArray(list) ? list : []))
      .catch((err) => setError(err.message || 'Failed to load'));
  }, []);

  async function save() {
    setStatus('');
    setError('');
    try {
      const token = localStorage.getItem('routerx_token') || '';
      await apiPost('/admin/model-pricing', { model, price_per_1k_usd: Number(price) }, token);
      const updated = items.filter((i) => i.model !== model).concat([{ model, price_per_1k_usd: Number(price) }]);
      setItems(updated);
      setModel('');
      setPrice('');
      setStatus('Saved');
    } catch (err: any) {
      setError(err.message || 'Failed to save');
    }
  }

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Model Pricing</h1>
            <p className="text-sm text-black/60">Configure USD per 1K tokens by model.</p>
          </div>
          <Nav />
        </div>
        {error && <p className="text-red-500">{error}</p>}
        {status && <p className="text-green-600">{status}</p>}

        <section className="card p-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <label className="block text-sm">
              Model
              <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" value={model} onChange={(e) => setModel(e.target.value)} placeholder="gpt-4o-mini" />
            </label>
            <label className="block text-sm">
              Price per 1K USD
              <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" value={price} onChange={(e) => setPrice(e.target.value)} placeholder="0.0015" />
            </label>
            <div className="flex items-end">
              <button className="px-3 py-2 rounded-lg bg-ink text-white w-full" onClick={save}>Save</button>
            </div>
          </div>
        </section>

        <section className="card p-4">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b">
                <th className="py-2">Model</th>
                <th>Price per 1K USD</th>
              </tr>
            </thead>
            <tbody>
              {items.map((i) => (
                <tr key={i.model} className="border-b last:border-0">
                  <td className="py-2">{i.model}</td>
                  <td>{Number(i.price_per_1k_usd).toFixed(6)}</td>
                </tr>
              ))}
              {!items.length && (
                <tr>
                  <td className="py-2 text-black/50" colSpan={2}>No pricing configured.</td>
                </tr>
              )}
            </tbody>
          </table>
        </section>
      </div>
    </main>
  );
}
