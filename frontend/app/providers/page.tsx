'use client';

import { useEffect, useMemo, useState } from 'react';
import Nav from '@/components/Nav';
import { apiGet, apiPut, apiPost } from '@/lib/api';

const PROVIDER_LABELS: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  'generic-openai': 'Generic OpenAI-Compatible'
};

const MODEL_OPTIONS: Record<string, string[]> = {
  openai: ['gpt-4o-mini', 'gpt-4o', 'gpt-4.1-mini', 'gpt-4.1', 'gpt-3.5-turbo'],
  anthropic: ['claude-3-5-sonnet', 'claude-3-5-haiku', 'claude-3-opus'],
  gemini: ['gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-2.5-flash', 'gemini-1.0-pro'],
  'generic-openai': ['custom-model']
};

export default function ProvidersPage() {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [selectedId, setSelectedId] = useState<string>('');

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
  }, [selectedId]);

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
        default_model: 'custom-model',
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
                  <div className="text-sm font-medium">{p.name || PROVIDER_LABELS[p.type] || p.type}</div>
                  <div className="text-xs text-black/50">{p.type}</div>
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
                  <button className="px-3 py-2 rounded-lg bg-ink text-white" onClick={() => saveProvider(selected)}>
                    Save
                  </button>
                </div>

                <label className="block text-sm">
                  Provider Name
                  <input
                    className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2"
                    value={selected.name || ''}
                    onChange={(e) => updateField(selected.id, 'name', e.target.value)}
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
                  />
                </label>

                <label className="block text-sm">
                  API Key (stored in DB)
                  <input
                    className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2"
                    value={selected.api_key || ''}
                    onChange={(e) => updateField(selected.id, 'api_key', e.target.value)}
                    placeholder="paste key"
                  />
                </label>

                <label className="block text-sm">
                  Default Model
                  <input
                    className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2"
                    value={selected.default_model || ''}
                    list={`models-${selected.id}`}
                    onChange={(e) => updateField(selected.id, 'default_model', e.target.value)}
                  />
                  <datalist id={`models-${selected.id}`}>
                    {(MODEL_OPTIONS[selected.type] || []).map((m) => (
                      <option key={m} value={m} />
                    ))}
                  </datalist>
                </label>

                <div className="flex gap-4 text-sm">
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={!!selected.supports_text}
                      onChange={(e) => updateField(selected.id, 'supports_text', e.target.checked)}
                    />
                    Text
                  </label>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={!!selected.supports_vision}
                      onChange={(e) => updateField(selected.id, 'supports_vision', e.target.checked)}
                    />
                    Vision
                  </label>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={!!selected.enabled}
                      onChange={(e) => updateField(selected.id, 'enabled', e.target.checked)}
                    />
                    Enabled
                  </label>
                </div>
              </div>
            )}
          </section>
        </div>
      </div>
    </main>
  );
}
