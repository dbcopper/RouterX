'use client';

import { useState } from 'react';
import { apiPost } from '@/lib/api';
import Link from 'next/link';
import { useRouter } from 'next/navigation';

export default function LoginPage() {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin123');
  const [tenantName, setTenantName] = useState('My Workspace');
  const [message, setMessage] = useState('');
  const router = useRouter();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMessage('');
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
        setMessage('Registration successful. Please sign in.');
        setMode('login');
      }
    } catch (err: any) {
      setMessage(err.message || 'Request failed');
    }
  }

  return (
    <main className="min-h-screen p-10 bg-gradient-to-br from-sand via-white to-sand">
      <div className="max-w-md mx-auto card p-8">
        <h1 className="text-2xl font-semibold">RouterX Login</h1>
        <div className="mt-4 flex gap-2">
          <button className={`px-3 py-1 rounded ${mode === 'login' ? 'bg-ink text-white' : 'border border-black/10'}`} onClick={() => setMode('login')}>Login</button>
          <button className={`px-3 py-1 rounded ${mode === 'register' ? 'bg-ink text-white' : 'border border-black/10'}`} onClick={() => setMode('register')}>Register</button>
        </div>
        <form onSubmit={onSubmit} className="mt-6 space-y-4">
          <label className="block text-sm">
            Username
            <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" value={username} onChange={(e) => setUsername(e.target.value)} />
          </label>
          <label className="block text-sm">
            Password
            <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </label>
          {mode === 'register' && (
            <label className="block text-sm">
              Workspace Name
              <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" value={tenantName} onChange={(e) => setTenantName(e.target.value)} />
            </label>
          )}
          <button className="w-full bg-ink text-white rounded-lg py-2" type="submit">{mode === 'login' ? 'Sign In' : 'Create Account'}</button>
        </form>
        {message && <p className="mt-4 text-sm text-black/70">{message}</p>}
        <div className="mt-4 text-sm">
          <Link href="/" className="underline">Back to Home</Link>
        </div>
      </div>
    </main>
  );
}
