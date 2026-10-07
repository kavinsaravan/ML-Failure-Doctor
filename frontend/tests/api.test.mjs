import test from 'node:test';
import assert from 'node:assert/strict';
import { api } from '../lib/api.ts';

test('protected operations send the supplied key and expired keys are cleared', async () => {
  const originalFetch=globalThis.fetch;
  const originalWindow=globalThis.window;
  const requests=[];
  const window=new EventTarget(); globalThis.window=window;
  let expired=0; window.addEventListener('crashlens-auth-required',()=>expired++);
  try {
    globalThis.fetch=async (url,options) => { requests.push({url,options}); return new Response(null,{status:204}); };
    api.setAPIKey('test-key'); await api.clearAllWorkloads();
    assert.equal(requests[0].options.headers.Authorization,'Bearer test-key');
    globalThis.fetch=async ()=>new Response('Invalid API key',{status:401});
    await assert.rejects(api.getWorkloads(), /Invalid API key/); assert.equal(expired,1);
    globalThis.fetch=async (url,options)=>{ requests.push({url,options}); return new Response('[]',{status:200}); };
    await api.getWorkloads(); assert.equal(requests.at(-1).options.headers.Authorization,undefined);
  } finally { globalThis.fetch=originalFetch; globalThis.window=originalWindow; api.setAPIKey(''); }
});


test('Fireworks credentials are diagnosis-only and cleared on workspace change', async () => {
  const originalFetch=globalThis.fetch;
  const requests=[];
  try {
    globalThis.fetch=async (url,options) => { requests.push({url,options}); return new Response('{}',{status:200}); };
    api.setAPIKey('alice'); api.setFireworksCredentials('fireworks-secret','user-model');
    await api.getWorkloads();
    assert.equal(requests.at(-1).options.headers['X-Fireworks-API-Key'],undefined);
    await api.diagnoseWorkload('1',true);
    assert.equal(requests.at(-1).options.headers['X-Fireworks-API-Key'],'fireworks-secret');
    assert.equal(requests.at(-1).options.headers['X-Fireworks-Model'],'user-model');
    api.setAPIKey('bob'); await api.diagnoseWorkload('2');
    assert.equal(requests.at(-1).options.headers['X-Fireworks-API-Key'],undefined);
  } finally { globalThis.fetch=originalFetch; api.setAPIKey(''); api.clearFireworksCredentials(); }
});
