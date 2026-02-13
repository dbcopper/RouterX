'use client';

import { usePathname } from 'next/navigation';
import Sidebar from './Sidebar';

const noSidebarPaths = ['/login', '/user-login', '/user/dashboard', '/user'];

export default function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const hideSidebar = noSidebarPaths.some((p) => pathname === p || pathname.startsWith(p + '/'));

  if (hideSidebar) {
    return <>{children}</>;
  }

  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-56">
        {children}
      </div>
    </div>
  );
}
