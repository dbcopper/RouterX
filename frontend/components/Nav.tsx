'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

const links = [
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/providers', label: 'Providers' },
  { href: '/routing', label: 'Routing' },
  { href: '/pricing', label: 'Pricing' },
  { href: '/tenants', label: 'Tenants' },
  { href: '/requests', label: 'Requests' }
];

export default function Nav() {
  const pathname = usePathname();

  function logout() {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('routerx_token');
      window.location.href = '/login';
    }
  }

  return (
    <div className="flex items-center gap-2 text-sm">
      {links.map((l) => {
        const active = pathname === l.href;
        return (
          <Link
            key={l.href}
            href={l.href}
            className={`px-3 py-1.5 rounded-full border transition-colors ${
              active
                ? 'border-black bg-black text-white font-medium'
                : 'border-black/10 bg-white/80 hover:bg-black/5'
            }`}
          >
            {l.label}
          </Link>
        );
      })}
      <button
        onClick={logout}
        className="px-3 py-1.5 rounded-full border border-black/10 bg-white/80 hover:bg-black/5 ml-2"
      >
        Logout
      </button>
    </div>
  );
}
