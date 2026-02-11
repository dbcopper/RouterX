'use client';

import Link from 'next/link';

const links = [
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/providers', label: 'Providers' },
  { href: '/pricing', label: 'Pricing' },
  { href: '/tenants', label: 'Tenants' },
  { href: '/requests', label: 'Requests' }
];

export default function Nav() {
  function logout() {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('routerx_token');
      window.location.href = '/login';
    }
  }

  return (
    <div className="flex items-center gap-3 text-sm">
      {links.map((l) => (
        <Link key={l.href} href={l.href} className="px-3 py-1 rounded-full border border-black/10 bg-white/80">
          {l.label}
        </Link>
      ))}
      <button onClick={logout} className="px-3 py-1 rounded-full border border-black/10 bg-white/80">
        Logout
      </button>
    </div>
  );
}
