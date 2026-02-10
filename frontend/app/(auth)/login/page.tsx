'use client';

import { useState } from 'react';
import { apiPost } from '@/lib/api';
import Link from 'next/link';

export default function LoginPage() {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin123');
  const [message, setMessage] = useState('');

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMessage('');
    try {
      const res = await apiPost('/admin/login', { username, password });
      localStorage.setItem('routerx_token', res.token);
      setMessage('Login successful. Token stored in localStorage.');
    } catch (err: any) {
      setMessage(err.message || 'Login failed');
    }
  }

  return (
    <main className="min-h-screen p-10 bg-gradient-to-br from-sand via-white to-sand">
      <div className="max-w-md mx-auto card p-8">
        <h1 className="text-2xl font-semibold">Admin Login</h1>
        <form onSubmit={onSubmit} className="mt-6 space-y-4">
          <label className="block text-sm">
            Username
            <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" value={username} onChange={(e) => setUsername(e.target.value)} />
          </label>
          <label className="block text-sm">
            Password
            <input className="mt-1 w-full border border-black/10 rounded-lg px-3 py-2" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </label>
          <button className="w-full bg-ink text-white rounded-lg py-2" type="submit">Sign In</button>
        </form>
        {message && <p className="mt-4 text-sm text-black/70">{message}</p>}
        <div className="mt-4 text-sm">
          <Link href="/dashboard" className="underline">Go to Dashboard</Link>
        </div>
      </div>
    </main>
  );
}
