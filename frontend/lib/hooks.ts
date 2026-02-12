'use client';
import { useState, useEffect, useCallback } from 'react';
import { apiGet } from './api';

export function useAdminFetch<T>(path: string, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refetch = useCallback(() => {
    const token = localStorage.getItem('routerx_token');
    if (!token) { setLoading(false); return; }
    setLoading(true);
    setError('');
    apiGet(path, token)
      .then((d) => setData(d))
      .catch((e) => setError(e.message || 'Request failed'))
      .finally(() => setLoading(false));
  }, [path, ...deps]);

  useEffect(() => { refetch(); }, [refetch]);

  return { data, loading, error, refetch };
}

export function useUserFetch<T>(path: string, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refetch = useCallback(() => {
    const token = localStorage.getItem('routerx_user_token');
    if (!token) { setLoading(false); return; }
    setLoading(true);
    setError('');
    apiGet(path, token)
      .then((d) => setData(d))
      .catch((e) => setError(e.message || 'Request failed'))
      .finally(() => setLoading(false));
  }, [path, ...deps]);

  useEffect(() => { refetch(); }, [refetch]);

  return { data, loading, error, refetch };
}
