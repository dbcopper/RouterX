'use client';

import { useEffect, useState } from 'react';
import Nav from '@/components/Nav';
import ConfirmModal from '@/components/ConfirmModal';
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api';

interface RoutingRule {
  id: string;
  tenant_id: string;
  capability: string;
  primary_provider_id: string;
  secondary_provider_id: string;
  model: string;
}

interface Provider {
  id: string;
  name: string;
  type: string;
  enabled: boolean;
  supports_text: boolean;
  supports_vision: boolean;
}

interface Tenant {
  id: string;
  name: string;
}

const emptyForm = { capability: 'text', primary_provider_id: '', secondary_provider_id: '', model: '' };

export default function RoutingPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [selectedTenant, setSelectedTenant] = useState('');
  const [rules, setRules] = useState<RoutingRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Form state
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);

  // Delete confirm
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const token = typeof window !== 'undefined' ? localStorage.getItem('routerx_token') || '' : '';

  useEffect(() => {
    Promise.all([apiGet('/admin/tenants', token), apiGet('/admin/providers', token)])
      .then(([t, p]) => {
        setTenants(Array.isArray(t) ? t : []);
        setProviders(Array.isArray(p) ? p : []);
      })
      .catch((e) => setError(e.message));
  }, []);

  useEffect(() => {
    if (!selectedTenant) { setRules([]); return; }
    setLoading(true);
    apiGet(`/admin/routing-rules?tenant_id=${selectedTenant}`, token)
      .then((r) => setRules(Array.isArray(r) ? r : []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [selectedTenant]);

  function refreshRules() {
    if (!selectedTenant) return;
    apiGet(`/admin/routing-rules?tenant_id=${selectedTenant}`, token)
      .then((r) => setRules(Array.isArray(r) ? r : []));
  }

  function providerName(id: string) {
    return providers.find((p) => p.id === id)?.name || id;
  }

  function enabledProviders() {
    return providers.filter((p) => p.enabled);
  }

  function openCreate() {
    setEditingId(null);
    setForm(emptyForm);
    setShowForm(true);
  }

  function openEdit(rule: RoutingRule) {
    setEditingId(rule.id);
    setForm({
      capability: rule.capability,
      primary_provider_id: rule.primary_provider_id,
      secondary_provider_id: rule.secondary_provider_id,
      model: rule.model
    });
    setShowForm(true);
  }

  async function saveRule() {
    setSaving(true);
    setError('');
    try {
      if (editingId) {
        await apiPut(`/admin/routing-rules/${editingId}`, { ...form, tenant_id: selectedTenant }, token);
      } else {
        await apiPost('/admin/routing-rules', { ...form, tenant_id: selectedTenant }, token);
      }
      setShowForm(false);
      refreshRules();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    try {
      await apiDelete(`/admin/routing-rules/${deleteTarget}`, token);
      setDeleteTarget(null);
      refreshRules();
    } catch (e: any) {
      setError(e.message);
    }
  }

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Routing Rules</h1>
            <p className="text-sm text-black/50">Configure provider routing and fallback per tenant</p>
          </div>
          <Nav />
        </div>

        {error && (
          <div className="card p-3 border-red-200 bg-red-50 text-sm text-red-600 flex items-center justify-between">
            <span>{error}</span>
            <button onClick={() => setError('')} className="text-red-400">✕</button>
          </div>
        )}

        {/* Tenant Selector */}
        <div className="card p-4">
          <div className="flex items-center gap-4">
            <label className="text-sm font-medium">Select Tenant</label>
            <select
              value={selectedTenant}
              onChange={(e) => setSelectedTenant(e.target.value)}
              className="px-3 py-1.5 text-sm border border-black/10 rounded-lg bg-white min-w-[280px]"
            >
              <option value="">Choose a tenant...</option>
              {tenants.map((t) => (
                <option key={t.id} value={t.id}>{t.name} ({t.id.slice(0, 8)}...)</option>
              ))}
            </select>
            {selectedTenant && (
              <button
                onClick={openCreate}
                className="ml-auto px-4 py-1.5 text-sm rounded-lg bg-black text-white hover:bg-black/80"
              >
                Add Rule
              </button>
            )}
          </div>
        </div>

        {/* Rules Table */}
        {selectedTenant && (
          <div className="card overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-black/[0.03] text-xs font-medium text-black/60 uppercase tracking-wide">
                  <th className="px-4 py-3 text-left">Capability</th>
                  <th className="px-4 py-3 text-left">Primary Provider</th>
                  <th className="px-4 py-3 text-left">Fallback Provider</th>
                  <th className="px-4 py-3 text-left">Model</th>
                  <th className="px-4 py-3 text-center w-40">Actions</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={5} className="px-4 py-8 text-center text-black/40">Loading...</td></tr>
                ) : rules.length === 0 ? (
                  <tr><td colSpan={5} className="px-4 py-8 text-center text-black/40">No routing rules configured for this tenant</td></tr>
                ) : rules.map((r) => (
                  <tr key={r.id} className="border-t border-black/5 hover:bg-black/[0.02]">
                    <td className="px-4 py-3">
                      <span className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        r.capability === 'vision' ? 'bg-purple-50 text-purple-700' : 'bg-blue-50 text-blue-700'
                      }`}>
                        {r.capability}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-medium">{providerName(r.primary_provider_id)}</td>
                    <td className="px-4 py-3 text-black/60">
                      {r.secondary_provider_id ? providerName(r.secondary_provider_id) : <span className="text-black/30">None</span>}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">{r.model}</td>
                    <td className="px-4 py-3 text-center">
                      <button
                        onClick={() => openEdit(r)}
                        className="text-xs px-3 py-1 rounded-lg border border-black/10 hover:bg-black/5 mr-2"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => setDeleteTarget(r.id)}
                        className="text-xs px-3 py-1 rounded-lg border border-red-200 text-red-500 hover:bg-red-50"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {!selectedTenant && (
          <div className="card p-12 text-center">
            <p className="text-black/40">Select a tenant to view and manage routing rules</p>
          </div>
        )}
      </div>

      {/* Create/Edit Modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 max-w-md w-full mx-4">
            <h3 className="font-semibold text-lg mb-4">{editingId ? 'Edit Rule' : 'Add Routing Rule'}</h3>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium block mb-1">Capability</label>
                <select
                  value={form.capability}
                  onChange={(e) => setForm({ ...form, capability: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-black/10 rounded-lg"
                >
                  <option value="text">Text</option>
                  <option value="vision">Vision</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium block mb-1">Primary Provider</label>
                <select
                  value={form.primary_provider_id}
                  onChange={(e) => setForm({ ...form, primary_provider_id: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-black/10 rounded-lg"
                >
                  <option value="">Select provider...</option>
                  {enabledProviders().map((p) => (
                    <option key={p.id} value={p.id}>{p.name} ({p.type})</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium block mb-1">Fallback Provider <span className="text-black/40 font-normal">(optional)</span></label>
                <select
                  value={form.secondary_provider_id}
                  onChange={(e) => setForm({ ...form, secondary_provider_id: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-black/10 rounded-lg"
                >
                  <option value="">None</option>
                  {enabledProviders()
                    .filter((p) => p.id !== form.primary_provider_id)
                    .map((p) => (
                      <option key={p.id} value={p.id}>{p.name} ({p.type})</option>
                    ))}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium block mb-1">Default Model</label>
                <input
                  type="text"
                  value={form.model}
                  onChange={(e) => setForm({ ...form, model: e.target.value })}
                  placeholder="e.g. gpt-4.1-mini"
                  className="w-full px-3 py-2 text-sm border border-black/10 rounded-lg"
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowForm(false)}
                className="px-4 py-2 text-sm rounded-lg border border-black/10 hover:bg-black/5"
              >
                Cancel
              </button>
              <button
                onClick={saveRule}
                disabled={saving || !form.primary_provider_id || !form.model}
                className="px-4 py-2 text-sm rounded-lg bg-black text-white hover:bg-black/80 disabled:opacity-50"
              >
                {saving ? 'Saving...' : editingId ? 'Update' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirm */}
      <ConfirmModal
        open={!!deleteTarget}
        title="Delete Routing Rule"
        message="Are you sure you want to delete this routing rule? Requests for this tenant and capability will fail without a rule."
        confirmLabel="Delete"
        onConfirm={confirmDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </main>
  );
}
