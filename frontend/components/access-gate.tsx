 'use client';

import { FormEvent, ReactNode, useCallback, useEffect, useRef, useState } from 'react';
import { usePathname } from 'next/navigation';
import { api } from '@/lib/api';

type Access = 'checking' | 'public' | 'locked' | 'authenticated';
export default function AccessGate({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const attempt = useRef(0);
  const [access, setAccess] = useState<Access>('checking');
  const [key, setKey] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [fireworksKey, setFireworksKey] = useState('');
  const [fireworksModel, setFireworksModel] = useState('');
  const [fireworksReady, setFireworksReady] = useState(false);
  const saveFireworks = (event: FormEvent) => {
    event.preventDefault(); api.setFireworksCredentials(fireworksKey, fireworksModel);
    setFireworksKey(''); setFireworksReady(true);
  };
  const clearFireworks = () => { api.clearFireworksCredentials(); setFireworksKey(''); setFireworksModel(''); setFireworksReady(false); };
  const check = useCallback(async () => {
    const token = ++attempt.current;
    const apply = (next: Access) => { if (token === attempt.current) { setAccess(next); setError(''); } };
    try {
      const requirements = await api.getAccessRequirements();
      if (!requirements.auth_required && !api.hasAPIKey()) { apply('public'); return; }
      try { await api.checkAccess(); apply('authenticated'); }
      catch { apply(requirements.auth_required ? 'locked' : 'public'); }
    } catch {
      if (token === attempt.current) { setAccess('locked'); setError('Cannot connect to the backend.'); }
    }
  }, []);
  useEffect(() => {
    if (pathname === '/') return;
    const initial = setTimeout(() => void check(), 0);
    const onAuthRequired = () => { api.clearFireworksCredentials(); setFireworksReady(false); setFireworksKey(''); setFireworksModel(''); void check(); };
    window.addEventListener('crashlens-auth-required', onAuthRequired);
    return () => { clearTimeout(initial); window.removeEventListener('crashlens-auth-required', onAuthRequired); };
  }, [pathname, check]);
  const connect = async (event: FormEvent) => {
    event.preventDefault(); ++attempt.current; setBusy(true); setError('');
    try {
      await api.checkAccess(key); api.setAPIKey(key); setKey(''); setAccess('authenticated');
    } catch (err) { setError((err as Error).message); }
    finally { setBusy(false); }
  };
  const disconnect = () => { clearFireworks(); api.setAPIKey(''); setKey(''); setAccess('checking'); void check(); };
  if (pathname === '/') return children;
  if (access === 'checking') return <main className="min-h-screen bg-slate-950 text-white p-8">Connecting to CrashLens…</main>;
  return <>
    <section className="bg-slate-900 text-white border-b border-slate-700 p-4">
      {access === 'authenticated' ? <div className="flex items-center gap-4">
        <span>Connected to your private workspace</span>
        <button onClick={disconnect} className="text-blue-300">Disconnect</button>
      </div> : <form onSubmit={connect} className="flex flex-wrap items-center gap-3">
        <span>{access === 'public' ? 'Public access. Connect to manage workloads.' : 'Enter your CrashLens API key to access workloads.'}</span>
        <label htmlFor="crashlens-api-key" className="sr-only">API key</label>
        <input id="crashlens-api-key" type="password" value={key} onChange={e => setKey(e.target.value)} autoComplete="off" required className="bg-slate-800 border border-slate-600 rounded px-3 py-2" />
        <button disabled={busy} className="bg-blue-600 rounded px-4 py-2">{busy ? 'Connecting…' : 'Connect'}</button>
        <button type="button" onClick={() => { void check(); }} className="text-blue-300">Retry connection</button>
      </form>}
      {access === 'authenticated' && <div className="mt-4">
        {fireworksReady ? <div className="flex gap-3 items-center">
          <span>AI diagnoses use your Fireworks account ({fireworksModel}).</span>
          <button onClick={clearFireworks} className="text-blue-300">Remove Fireworks key</button>
        </div> : <form onSubmit={saveFireworks} className="flex flex-wrap items-center gap-3">
          <label htmlFor="fireworks-key">Your Fireworks API key</label>
          <input id="fireworks-key" type="password" value={fireworksKey} onChange={e => setFireworksKey(e.target.value)} autoComplete="off" required className="bg-slate-800 border border-slate-600 rounded px-3 py-2" />
          <label htmlFor="fireworks-model">Model identifier</label>
          <input id="fireworks-model" value={fireworksModel} onChange={e => setFireworksModel(e.target.value)} placeholder="accounts/fireworks/models/..." required className="bg-slate-800 border border-slate-600 rounded px-3 py-2" />
          <button className="bg-blue-600 rounded px-4 py-2">Use my Fireworks account</button>
          <p className="w-full text-sm text-slate-400">Optional. Choose a model with tool calling. Your key stays in memory for this session and is sent only for diagnosis. New diagnoses use your credits; saved reports are reused. Without a key, user workloads receive rule-based reports.</p>
        </form>}
      </div>}
      {error && <p role="alert" className="text-red-300 mt-2">{error}</p>}
    </section>
    {(access === 'public' || access === 'authenticated') && children}
  </>;
}
