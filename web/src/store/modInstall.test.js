import { createPinia, setActivePinia } from 'pinia';
import { expect, it, vi } from 'vitest';
import { useModInstallStore } from './modInstall';
vi.mock('../api', () => ({ default: {} }));
it('dismisses only finished jobs', () => {
    setActivePinia(createPinia());
    const store = useModInstallStore();
    store.jobs = ['running', 'completed', 'failed'].map(status => ({ jobId: status, status }));
    store.dismissJob('running');
    expect(store.jobs).toHaveLength(3);
    store.dismissJob('completed');
    store.dismissJob('failed');
    expect(store.jobs.map(job => job.jobId)).toEqual(['running']);
});
