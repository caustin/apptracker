import type { Db, Goals, Resume } from './types';

async function j<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) throw new Error(`${res.status}: ${await res.text()}`);
  return res.json();
}

const json = (method: string, body: unknown): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
});

export const api = {
  loadAll: async (): Promise<Db> => {
    const [positions, people, events, resumes, goals] = await Promise.all([
      j<Db['positions']>('/api/positions'),
      j<Db['people']>('/api/people'),
      j<Db['events']>('/api/events'),
      j<Db['resumes']>('/api/resumes'),
      j<Goals>('/api/goals'),
    ]);
    return { positions, people, events, resumes, goals };
  },
  uploadResume: (file: File, name?: string) => {
    const fd = new FormData();
    fd.append('file', file);
    if (name) fd.append('name', name);
    return j<Resume>('/api/resumes', { method: 'POST', body: fd });
  },
  create: <T>(resource: string, body: unknown) => j<T>(`/api/${resource}`, json('POST', body)),
  update: <T>(resource: string, id: number, body: unknown) =>
    j<T>(`/api/${resource}/${id}`, json('PUT', body)),
  remove: (resource: string, id: number) =>
    j<{ ok: boolean }>(`/api/${resource}/${id}`, { method: 'DELETE' }),
  saveGoals: (g: Partial<Goals>) => j<Goals>('/api/goals', json('PUT', g)),
};
