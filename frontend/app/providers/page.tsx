'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import Nav from '@/components/Nav';
import StatusBadge from '@/components/StatusBadge';
import { apiDelete, apiGet, apiPut, apiPost } from '@/lib/api';

interface ProviderHealth {
  provider_id: string;
  health_status: string;
  circuit_open: boolean;
  enabled: boolean;
}

const PROVIDER_LABELS: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  'generic-openai': 'Generic OpenAI-Compatible'
};

export default function ProvidersPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [selectedId, setSelectedId] = useState<string>('');
  const [models, setModels] = useState<string[]>([]);
  const [newModel, setNewModel] = useState('');
  const [healthMap, setHealthMap] = useState<Record<string, ProviderHealth>>({});
  const lastSavedKeyRef = useRef<{ id: string; key: string } | null>(null);

  const selected = useMemo(() => items.find((p) => p.id === selectedId) || items[0], [items, selectedId]);

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    apiGet('/admin/providers', token)
      .then((list) => {
        const safe = Array.isArray(list) ? list : [];
        setItems(safe);
        if (safe.length && !selectedId) setSelectedId(safe[0].id);
      })
      .catch((err) => setError(err.message || 'Failed to load'));
    apiGet('/admin/provider-health', token)
      .then((list) => {
        const map: Record<string, ProviderHealth> = {};
        if (Array.isArray(list)) list.forEach((h: ProviderHealth) => { map[h.provider_id] = h; });
        setHealthMap(map);
      })
      .catch(() => {});
  }, [selectedId]);

  useEffect(() => {
    const token = localStorage.getItem('routerx_token') || '';
    if (!selected?.type) {
      setModels([]);
      return;
    }
    apiGet(`/admin/models?provider_type=${encodeURIComponent(selected.type)}`, token)
      .then((list) => setModels(Array.isArray(list) ? list : []))
      .catch(() => setModels([]));
  }, [selected?.type]);

  useEffect(() => {
    if (!selected?.id) return;
    if (!selected.api_key) return;
    if (lastSavedKeyRef.current?.id === selected.id && lastSavedKeyRef.current?.key === selected.api_key) {
      return;
    }
    const timer = setTimeout(() => {
      saveProvider(selected).then(() => {
        lastSavedKeyRef.current = { id: selected.id, key: selected.api_key };
        updateField(selected.id, 'has_api_key', true);
        updateField(selected.id, 'api_key', '');
      });
    }, 600);
    return () => clearTimeout(timer);
  }, [selected?.id, selected?.api_key]);

  function updateField(id: string, key: string, value: any) {
    setItems((prev) => prev.map((p) => (p.id === id ? { ...p, [key]: value } : p)));
  }

  async function saveProvider(p: any) {
    setStatus('');
    setError('');
    try {
      const token = localStorage.getItem('routerx_token') || '';
      await apiPut(`/admin/providers/${p.id}`, {
        base_url: p.base_url || '',
        api_key: p.api_key || '',
        default_model: p.default_model || '',
        supports_text: !!p.supports_text,
        supports_vision: !!p.supports_vision,
        enabled: !!p.enabled
      }, token);
      setStatus(`Saved ${p.name}`);
    } catch (err: any) {
      setError(err.message || 'Failed to save');
    }
  }

  async function addGeneric() {
    setStatus('');
    setError('');
    try {
      const token = localStorage.getItem('routerx_token') || '';
      const created = await apiPost('/admin/providers', {
        name: `Generic ${items.filter((p) => p.type === 'generic-openai').length + 1}`,
        type: 'generic-openai',
        base_url: '',
        default_model: '',
        supports_text: true,
        supports_vision: false,
        enabled: true
      }, token);
      const next = [...items, created];
      setItems(next);
      setSelectedId(created.id);
      setStatus('Generic provider added');
    } catch (err: any) {
      setError(err.message || 'Failed to add');
    }
  }

  async function addModel() {
    if (!selected?.type || !newModel.trim()) return;
    setError('');
    setStatus('');
    try {
      const token = localStorage.getItem('routerx_token') || '';
      await apiPost('/admin/models', { model: newModel.trim(), provider_type: selected.type }, token);
      setModels((prev) => Array.from(new Set([...prev, newModel.trim()])).sort());
      setNewModel('');
      setStatus('Model added');
    } catch (err: any) {
      setError(err.message || 'Failed to add model');
    }
  }

  async function deleteModel(name: string) {
    setError('');
    setStatus('');
    try {
      const token = localStorage.getItem('routerx_token') || '';
      await apiDelete(`/admin/models/${encodeURIComponent(name)}`, token);
      setModels((prev) => prev.filter((m) => m !== name));
      setStatus('Model removed');
    } catch (err: any) {
      setError(err.message || 'Failed to delete model');
    }
  }

  return (
    <main className="min-h-screen p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold">Providers</h1>
            <p className="text-sm text-black/60">Select a provider and configure keys, models, and capabilities.</p>
          </div>
          <Nav />
        </div>

        {error && <p className="text-red-500">{error}</p>}
        {status && <p className="text-green-600">{status}</p>}

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <aside className="card p-4 md:col-span-1">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-semibold">Providers</h3>
              <button className="px-2 py-1 rounded bg-ink text-white text-xs" onClick={addGeneric}>+ Generic</button>
            </div>
            <div className="space-y-2">
              {items.map((p) => (
                <button
                  key={p.id}
                  onClick={() => setSelectedId(p.id)}
                  className={`w-full text-left px-3 py-2 rounded-lg border ${p.id === selected?.id ? 'border-ink bg-white' : 'border-black/10 bg-white/70'}`}
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{p.name || PROVIDER_LABELS[p.type] || p.type}</span>
                    {healthMap[p.id] && (
                      <StatusBadge
                        status={
                          !p.enabled ? 'inactive'
                          : healthMap[p.id]?.circuit_open ? 'fail'
                          : healthMap[p.id]?.health_status === 'ok' ? 'ok'
                          : healthMap[p.id]?.health_status === 'fail' ? 'fail'
                          : 'unknown'
                        }
                      />
                    )}
                  </div>
                  <div className="text-xs text-black/50">{p.type}{!p.enabled ? ' · disabled' : ''}</div>
                </button>
              ))}
            </div>
          </aside>

          <section className="card p-6 md:col-span-3">
            {!selected && <p className="text-sm">No provider selected.</p>}
            {selected && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="text-xl font-semibold">{selected.name || PROVIDER_LABELS[selected.type] || selected.type}</h2>
                    <p className="text-xs text-black/60">ID: {selected.id}</p>
                  </div>
                </div>

                <label className="block text-sm">
                  Provider Name
                  <input
                    className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2"
                    value={selected.name || ''}
                    onChange={(e) => updateField(selected.id, 'name', e.target.value)}
                    onBlur={() => saveProvider(selected)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        saveProvider(selected);
                      }
                    }}
                  />
                </label>

                <label className="block text-sm">
                  Base URL
                  <input
                    className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2"
                    value={selected.base_url || ''}
                    onChange={(e) => updateField(selected.id, 'base_url', e.target.value)}
                    placeholder={selected.type === 'generic-openai' ? 'https://api.example.com' : 'Managed by provider'}
                    disabled={selected.type !== 'generic-openai'}
                    onBlur={() => saveProvider(selected)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        saveProvider(selected);
                      }
                    }}
                  />
                </label>

                <label className="block text-sm">
                  API Key (stored in DB)
                  <div className="mt-1 flex gap-2">
                    <input
                      className="flex-1 border border-black/10 rounded-lg px-3 py-2"
                      value={selected.has_api_key ? '' : (selected.api_key || '')}
                      onChange={(e) => updateField(selected.id, 'api_key', e.target.value)}
                      placeholder={selected.has_api_key ? 'Saved. Use Clear to replace.' : 'paste key'}
                      disabled={!!selected.has_api_key}
                      onBlur={() => saveProvider(selected)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault();
                          saveProvider(selected);
                        }
                      }}
                    />
                    {selected.has_api_key && (
                      <button
                        className="px-3 py-2 rounded-lg border border-black/10 text-sm"
                        onClick={async () => {
                          try {
                            const token = localStorage.getItem('routerx_token') || '';
                            await apiDelete(`/admin/providers/${selected.id}/api-key`, token);
                            updateField(selected.id, 'api_key', '');
                            updateField(selected.id, 'has_api_key', false);
                            setStatus('API key cleared');
                          } catch (err: any) {
                            setError(err.message || 'Failed to clear key');
                          }
                        }}
                      >
                        Clear
                      </button>
                    )}
                  </div>
                </label>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-semibold">Models</h3>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      className="flex-1 border border-black/10 rounded-lg px-3 py-2 text-sm"
                      value={newModel}
                      onChange={(e) => setNewModel(e.target.value)}
                      placeholder="e.g. gemini-2.5-flash"
                    />
                    <button className="px-3 py-2 rounded-lg bg-ink text-white text-sm" onClick={addModel}>Add</button>
                  </div>
                  <div className="border rounded-lg overflow-hidden text-sm">
                    {!models.length && <div className="px-3 py-2 text-black/50">No models yet.</div>}
                    {models.map((m) => (
                      <div key={m} className="flex items-center justify-between px-3 py-2 border-t">
                        <span className="text-sm">{m}</span>
                        <button className="text-xs text-red-600 underline" onClick={() => deleteModel(m)}>Delete</button>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </section>
        </div>
      </div>
    </main>
  );
}
