'use client';

import { useEffect, useState } from 'react';

import { apiGet, apiPost, apiDelete } from '@/lib/api';

interface Webhook {
  id: number;
  url: string;
  events: string[];
  secret: string;
  enabled: boolean;
  created_at: string;
}

export default function WebhooksPage() {
  const [hooks, setHooks] = useState<Webhook[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAdd, setShowAdd] = useState(false);
  const [newUrl, setNewUrl] = useState('');
  const [newSecret, setNewSecret] = useState('');
  const [saving, setSaving] = useState(false);

  function token() {
    return typeof window !== 'undefined' ? localStorage.getItem('routerx_token') || '' : '';
  }

  async function load() {
    setLoading(true);
    try {
      const data = await apiGet('/admin/webhooks', token());
      setHooks(Array.isArray(data) ? data : []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function handleAdd() {
    if (!newUrl) return;
    setSaving(true);
    try {
      await apiPost('/admin/webhooks', {
        url: newUrl,
        events: ['request.completed'],
        secret: newSecret
      }, token());
      setShowAdd(false);
      setNewUrl('');
      setNewSecret('');
      load();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await apiDelete(`/admin/webhooks/${id}`, token());
      load();
    } catch (e: any) {
      setError(e.message);
    }
  }

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-semibold">Webhooks</h1>
          <p className="text-sm text-black/50">Receive real-time notifications for API events</p>
        </div>

        {error && (
          <div className="card p-3 border-red-200 bg-red-50 text-sm text-red-600">{error}</div>
        )}

        <div className="card p-4">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">Registered Webhooks</h2>
            <button
              onClick={() => setShowAdd(true)}
              className="px-4 py-2 text-sm rounded-lg bg-black text-white hover:bg-black/80"
            >
              Add Webhook
            </button>
          </div>

          {loading ? (
            <p className="text-sm text-black/40">Loading...</p>
          ) : hooks.length === 0 ? (
            <p className="text-sm text-black/40">No webhooks configured. Add one to receive event notifications.</p>
          ) : (
            <div className="space-y-3">
              {hooks.map((h) => (
                <div key={h.id} className="border border-black/10 rounded-lg p-4 flex items-center justify-between">
                  <div>
                    <p className="font-mono text-sm">{h.url}</p>
                    <div className="flex items-center gap-2 mt-1">
                      {h.events.map((e) => (
                        <span key={e} className="text-xs px-2 py-0.5 rounded-full bg-blue-50 text-blue-700">{e}</span>
                      ))}
                      {h.secret && <span className="text-xs text-black/40">signed</span>}
                      <span className="text-xs text-black/30">Added {new Date(h.created_at).toLocaleDateString()}</span>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDelete(h.id)}
                    className="text-xs text-red-400 hover:text-red-600 px-3 py-1"
                  >
                    Delete
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="card p-4">
          <h2 className="text-lg font-semibold mb-2">Event Types</h2>
          <div className="text-sm space-y-2">
            <div className="flex items-start gap-3">
              <code className="text-xs bg-black/5 px-2 py-1 rounded font-mono">request.completed</code>
              <span className="text-black/60">Fired after every API request completes. Includes tenant_id, provider, model, latency, tokens, cost, and status.</span>
            </div>
          </div>
        </div>

        <div className="card p-4">
          <h2 className="text-lg font-semibold mb-2">Signature Verification</h2>
          <p className="text-sm text-black/60">
            If a secret is configured, each webhook request includes an <code className="bg-black/5 px-1 rounded">X-RouterX-Signature</code> header
            containing an HMAC-SHA256 hex digest of the request body, signed with your secret.
          </p>
        </div>
      </div>

      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 max-w-md w-full mx-4">
            <h3 className="font-semibold text-lg mb-4">Add Webhook</h3>
            <label className="text-sm font-medium">URL</label>
            <input
              type="url"
              value={newUrl}
              onChange={(e) => setNewUrl(e.target.value)}
              placeholder="https://your-server.com/webhook"
              className="w-full mt-1 px-3 py-2 border border-black/10 rounded-lg text-sm"
              autoFocus
            />
            <label className="text-sm font-medium mt-3 block">Secret (optional)</label>
            <input
              type="text"
              value={newSecret}
              onChange={(e) => setNewSecret(e.target.value)}
              placeholder="Used for HMAC signature verification"
              className="w-full mt-1 px-3 py-2 border border-black/10 rounded-lg text-sm"
            />
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowAdd(false)} className="px-4 py-2 text-sm rounded-lg border border-black/10 hover:bg-black/5">Cancel</button>
              <button onClick={handleAdd} disabled={saving || !newUrl} className="px-4 py-2 text-sm rounded-lg bg-black text-white hover:bg-black/80 disabled:opacity-50">
                {saving ? 'Adding...' : 'Add'}
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
