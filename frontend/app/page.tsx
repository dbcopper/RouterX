'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiPost } from '@/lib/api';

const FEATURES = [
  { title: 'Auto-Routing', desc: 'Model name in, best provider out. Zero config.' },
  { title: 'Multi-Provider Fallback', desc: 'Circuit breaker + latency-aware failover.' },
  { title: '50+ Models', desc: 'OpenAI, Anthropic, Gemini, DeepSeek, Mistral.' },
  { title: 'Per-Tenant Billing', desc: 'Balance, spend limits, transaction ledger.' },
  { title: 'BYOK', desc: 'Bring your own API key per request.' },
  { title: 'Real-Time Webhooks', desc: 'HMAC-signed event notifications.' },
];

const PROVIDERS = ['OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Mistral'];

function FloatingOrb({ delay, size, x, y }: { delay: number; size: number; x: number; y: number }) {
  return (
    <div
      className="absolute rounded-full opacity-20 blur-2xl animate-float"
      style={{
        width: size,
        height: size,
        left: `${x}%`,
        top: `${y}%`,
        background: 'radial-gradient(circle, #0f6b4b 0%, transparent 70%)',
        animationDelay: `${delay}s`,
        animationDuration: `${6 + delay}s`,
      }}
    />
  );
}

function RoutingAnimation() {
  const [activeProvider, setActiveProvider] = useState(0);
  const [packets, setPackets] = useState<number[]>([]);

  useEffect(() => {
    const interval = setInterval(() => {
      setActiveProvider((p) => (p + 1) % PROVIDERS.length);
      setPackets((prev) => [...prev.slice(-4), Date.now()]);
    }, 2000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="relative w-full h-full flex items-center justify-center">
      {/* Center hub */}
      <div className="absolute z-10 w-20 h-20 rounded-2xl bg-ink shadow-2xl shadow-ink/30 flex items-center justify-center">
        <span className="text-white font-bold text-lg">RX</span>
      </div>

      {/* Request beam from left */}
      <div className="absolute left-[5%] top-1/2 -translate-y-1/2 flex items-center gap-3">
        <div className="text-xs font-mono text-black/40 text-right w-20">
          /v1/chat<br />completions
        </div>
        <div className="relative w-[calc(100%-5rem)]">
          {packets.map((id) => (
            <div
              key={id}
              className="absolute top-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-leaf animate-packet-in"
            />
          ))}
          <div className="w-24 h-[2px] bg-gradient-to-r from-leaf/60 to-leaf/20" />
        </div>
      </div>

      {/* Provider nodes */}
      {PROVIDERS.map((name, i) => {
        const angle = -60 + (i * 30);
        const rad = (angle * Math.PI) / 180;
        const radius = 140;
        const cx = Math.cos(rad) * radius;
        const cy = Math.sin(rad) * radius;
        const isActive = i === activeProvider;

        return (
          <div
            key={name}
            className="absolute z-20 transition-all duration-500"
            style={{
              transform: `translate(${cx + 160}px, ${cy}px)`,
            }}
          >
            {/* Connection line */}
            <div
              className={`absolute right-full top-1/2 h-[2px] w-12 -translate-y-1/2 transition-all duration-500 ${
                isActive ? 'bg-gradient-to-r from-leaf/20 to-leaf' : 'bg-black/5'
              }`}
            />
            {/* Node */}
            <div
              className={`px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-all duration-500 border ${
                isActive
                  ? 'bg-leaf/10 border-leaf/30 text-leaf scale-110 shadow-lg shadow-leaf/10'
                  : 'bg-white/80 border-black/10 text-black/40'
              }`}
            >
              {name}
              {isActive && (
                <span className="ml-1.5 inline-block w-1.5 h-1.5 rounded-full bg-leaf animate-pulse" />
              )}
            </div>
          </div>
        );
      })}

      {/* Status text */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2 text-[10px] font-mono text-black/30 whitespace-nowrap animate-pulse">
        routing to {PROVIDERS[activeProvider]}...
      </div>
    </div>
  );
}

export default function LandingPage() {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [tenantName, setTenantName] = useState('');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [mounted, setMounted] = useState(false);
  const router = useRouter();

  useEffect(() => { setMounted(true); }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMessage('');
    setLoading(true);
    try {
      if (mode === 'login') {
        const res = await apiPost('/auth/login', { username, password });
        if (res.role === 'admin') {
          localStorage.setItem('routerx_token', res.token);
          router.push('/dashboard');
        } else {
          localStorage.setItem('routerx_user_token', res.token);
          router.push('/user/dashboard');
        }
      } else {
        await apiPost('/auth/register', { username, password, tenant_name: tenantName });
        setMessage('Account created! Please sign in.');
        setMode('login');
      }
    } catch (err: any) {
      setMessage(err.message || 'Request failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen bg-sand relative overflow-hidden">
      {/* Background orbs */}
      <FloatingOrb delay={0} size={300} x={10} y={20} />
      <FloatingOrb delay={2} size={200} x={70} y={60} />
      <FloatingOrb delay={4} size={250} x={50} y={10} />

      <div className="relative z-10 min-h-screen flex items-center justify-center gap-12 lg:gap-20 px-8 lg:px-16 max-w-7xl mx-auto">
        {/* Left: Hero + Animation */}
        <div className={`flex-1 flex flex-col justify-center max-w-xl transition-all duration-1000 ${mounted ? 'opacity-100 translate-x-0' : 'opacity-0 -translate-x-10'}`}>
          {/* Logo */}
          <div className="flex items-center gap-3 mb-10">
            <div className="w-10 h-10 rounded-xl bg-ink flex items-center justify-center shadow-lg shadow-ink/20">
              <span className="text-white font-bold">RX</span>
            </div>
            <span className="text-2xl font-semibold tracking-tight">RouterX</span>
          </div>

          {/* Headline */}
          <h1 className="text-4xl lg:text-5xl font-semibold leading-tight tracking-tight">
            One API.<br />
            <span className="text-leaf">Every model.</span><br />
            Zero config.
          </h1>
          <p className="mt-4 text-lg text-black/50 max-w-lg">
            Production-grade LLM gateway with auto-routing, billing, and observability. Drop-in OpenAI-compatible replacement.
          </p>

          {/* Routing animation */}
          <div className="mt-8 h-56 w-full max-w-lg relative">
            <RoutingAnimation />
          </div>

          {/* Feature pills */}
          <div className="mt-6 flex flex-wrap gap-2 max-w-lg">
            {FEATURES.map((f) => (
              <div
                key={f.title}
                className="group relative px-3 py-1.5 rounded-full border border-black/10 bg-white/60 text-xs text-black/60 hover:border-leaf/30 hover:text-leaf transition-colors cursor-default"
              >
                {f.title}
                <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-3 py-1.5 rounded-lg bg-ink text-white text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                  {f.desc}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Right: Login card */}
        <div className={`w-full max-w-sm flex items-center transition-all duration-1000 delay-300 ${mounted ? 'opacity-100 translate-x-0' : 'opacity-0 translate-x-10'}`}>
          <div className="w-full bg-white/80 backdrop-blur-xl border border-black/10 rounded-2xl shadow-xl shadow-black/5 p-8">
            <h2 className="text-xl font-semibold">
              {mode === 'login' ? 'Welcome back' : 'Create account'}
            </h2>
            <p className="text-sm text-black/40 mt-1">
              {mode === 'login' ? 'Sign in to your RouterX console' : 'Get started with your own workspace'}
            </p>

            <div className="mt-5 flex gap-1 p-1 bg-black/5 rounded-lg">
              <button
                onClick={() => setMode('login')}
                className={`flex-1 py-1.5 text-sm rounded-md transition-all ${
                  mode === 'login' ? 'bg-white shadow-sm font-medium' : 'text-black/50 hover:text-black'
                }`}
              >
                Sign In
              </button>
              <button
                onClick={() => setMode('register')}
                className={`flex-1 py-1.5 text-sm rounded-md transition-all ${
                  mode === 'register' ? 'bg-white shadow-sm font-medium' : 'text-black/50 hover:text-black'
                }`}
              >
                Register
              </button>
            </div>

            <form onSubmit={onSubmit} className="mt-5 space-y-3">
              <label className="block">
                <span className="text-sm font-medium text-black/70">Username</span>
                <input
                  className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2.5 text-sm bg-white focus:border-leaf/50 focus:ring-1 focus:ring-leaf/20 outline-none transition-colors"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="admin"
                  autoComplete="username"
                />
              </label>
              <label className="block">
                <span className="text-sm font-medium text-black/70">Password</span>
                <input
                  className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2.5 text-sm bg-white focus:border-leaf/50 focus:ring-1 focus:ring-leaf/20 outline-none transition-colors"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="admin123"
                  autoComplete="current-password"
                />
              </label>
              {mode === 'register' && (
                <label className="block animate-fade-in">
                  <span className="text-sm font-medium text-black/70">Workspace Name</span>
                  <input
                    className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2.5 text-sm bg-white focus:border-leaf/50 focus:ring-1 focus:ring-leaf/20 outline-none transition-colors"
                    value={tenantName}
                    onChange={(e) => setTenantName(e.target.value)}
                    placeholder="My Company"
                  />
                </label>
              )}
              <button
                type="submit"
                disabled={loading}
                className="w-full bg-ink text-white rounded-lg py-2.5 text-sm font-medium hover:bg-ink/90 disabled:opacity-50 transition-colors mt-2"
              >
                {loading ? (
                  <span className="flex items-center justify-center gap-2">
                    <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    {mode === 'login' ? 'Signing in...' : 'Creating...'}
                  </span>
                ) : (
                  mode === 'login' ? 'Sign In' : 'Create Account'
                )}
              </button>
            </form>

            {message && (
              <p className={`mt-3 text-sm ${message.includes('created') || message.includes('successful') ? 'text-leaf' : 'text-red-500'}`}>
                {message}
              </p>
            )}

            <div className="mt-5 pt-4 border-t border-black/5 text-center">
              <p className="text-xs text-black/30">
                Default: admin / admin123
              </p>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
