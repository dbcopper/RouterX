'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet } from '@/lib/api';
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

export default function DashboardPage() {
  const [data, setData] = useState<any[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/requests', token)
      .then((logs) => {
        const list = Array.isArray(logs) ? logs : [];
        const mapped = list.slice(0, 10).map((l: any, idx: number) => ({
          name: `-${idx + 1}`,
          latency: l.latency_ms || 0,
          tokens: l.tokens || 0
        }));
        setData(mapped.reverse());
      })
      .catch((err) => setError(err.message || 'Failed to load'));
  }, []);

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Dashboard</h1>
            <p className="text-sm text-black/60">Multi-provider routing health at a glance.</p>
          </div>
          <Nav />
        </div>

        <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="card p-4">
            <p className="text-xs text-black/60">Request Volume</p>
            <p className="text-2xl font-semibold">Live</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-black/60">Error Rate</p>
            <p className="text-2xl font-semibold">Tracked</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-black/60">P95 Latency</p>
            <p className="text-2xl font-semibold">Streaming</p>
          </div>
        </section>

        <section className="card p-6">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">Recent Request Latency</h2>
            {error && <span className="text-sm text-red-500">{error}</span>}
          </div>
          <div className="h-64 mt-4">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={data}>
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Area type="monotone" dataKey="latency" stroke="#0f6b4b" fill="#a7f3d0" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </section>
      </div>
    </main>
  );
}
