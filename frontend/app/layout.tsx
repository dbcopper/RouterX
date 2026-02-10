import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'RouterX Admin',
  description: 'Provider-agnostic LLM/VLM gateway admin console'
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        {children}
      </body>
    </html>
  );
}
