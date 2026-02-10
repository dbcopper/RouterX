import Link from 'next/link';

const links = [
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/providers', label: 'Providers' },
  { href: '/routing', label: 'Routing Rules' },
  { href: '/tenants', label: 'Tenants' },
  { href: '/requests', label: 'Requests' }
];

export default function Nav() {
  return (
    <nav className="flex flex-wrap gap-3 text-sm">
      {links.map((l) => (
        <Link key={l.href} href={l.href} className="px-3 py-1 rounded-full border border-black/10 bg-white/80">
          {l.label}
        </Link>
      ))}
    </nav>
  );
}
