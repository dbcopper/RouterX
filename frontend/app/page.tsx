import Link from 'next/link';

export default function Home() {
  return (
    <main className="min-h-screen bg-gradient-to-br from-sand via-white to-sand p-10">
      <div className="max-w-3xl mx-auto card p-8">
        <h1 className="text-3xl font-semibold">RouterX Admin Console</h1>
        <p className="mt-3 text-sm text-black/70">
          Manage providers, routing rules, tenants, and request telemetry for your multi-provider gateway.
        </p>
        <div className="mt-6 flex gap-3">
          <Link className="px-4 py-2 rounded-lg bg-ink text-white" href="/login">Login</Link>
          <Link className="px-4 py-2 rounded-lg border border-black/10" href="/dashboard">Admin Dashboard</Link>
        </div>
      </div>
    </main>
  );
}
