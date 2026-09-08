import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import ModpackFileBrowser from './ModpackFileBrowser.vue';

const entries = [
  { id: 'common:a', path: 'config/a.toml', size: 2048, editable: true, source: '共用覆寫' },
  { id: 'server:a', path: 'config/a.toml', size: 1024, editable: true, source: '伺服器覆寫' },
  { id: 'b', path: 'config/sub/b.json', size: 4096 },
  { id: 'blocked', path: 'config/client.json', size: 1024, reason: '僅客戶端' },
  { id: 'mod', path: 'mods/test.jar', size: 16384 },
];
const checkbox = (wrapper, name) => wrapper.get(`[role="checkbox"][aria-label="${name}"]`);
const open = (wrapper, name) => wrapper.get(`button[aria-label="開啟資料夾 ${name}"]`).trigger('click');
const update = wrapper => wrapper.emitted('update:selected').at(-1)[0];

describe('modpack folder selection', () => {
  it('shows folder totals, navigates down/back and preserves both override layers', async () => {
    const wrapper = mount(ModpackFileBrowser, { props: { entries, selected: ['common:a'] } });
    expect(wrapper.text()).toContain('8.00 KiB');
    expect(wrapper.text()).toContain('16.00 KiB');
    expect(wrapper.text()).not.toContain('修改時間');
    await open(wrapper, 'config');
    expect(wrapper.text()).toContain('共用覆寫');
    expect(wrapper.text()).toContain('伺服器覆寫');
    expect(wrapper.findAll('button[aria-label="編輯 a.toml"]')).toHaveLength(2);
    await wrapper.findAll('button[aria-label="編輯 a.toml"]')[1].trigger('click');
    expect(wrapper.emitted('edit')[0][0].id).toBe('server:a');
    await open(wrapper, 'sub');
    expect(wrapper.text()).toContain('b.json');
    await wrapper.findAll('button').find(button => button.text() === '父資料夾').trigger('click');
    expect(wrapper.text()).toContain('a.toml');
    await wrapper.findAll('button').find(button => button.text() === '父資料夾').trigger('click');
    expect(wrapper.text()).toContain('mods');
    wrapper.unmount();
  });
  it('selects descendants, excludes disabled files and leaves other folders unchanged', async () => {
    const wrapper = mount(ModpackFileBrowser, { props: { entries, selected: ['common:a', 'mod'] } });
    expect(checkbox(wrapper, '選取 config').attributes('aria-checked')).toBe('mixed');
    await checkbox(wrapper, '選取 config').trigger('click');
    expect(new Set(update(wrapper))).toEqual(new Set(['common:a', 'server:a', 'b', 'mod']));
    await wrapper.setProps({ selected: update(wrapper) });
    expect(checkbox(wrapper, '選取 config').attributes('aria-checked')).toBe('true');
    await open(wrapper, 'config');
    await checkbox(wrapper, '選取目前列表的所有可用檔案').trigger('click');
    expect(update(wrapper)).toEqual(['mod']);
    const updates = wrapper.emitted('update:selected').length;
    await checkbox(wrapper, '選取 client.json').trigger('click');
    expect(wrapper.emitted('update:selected')).toHaveLength(updates);
    wrapper.unmount();
  });
  it('updates totals after edits and sorts by size', async () => {
    const wrapper = mount(ModpackFileBrowser, { props: { entries, selected: [] } });
    await wrapper.setProps({ entries: entries.map(file => file.id === 'common:a' ? { ...file, size: 32768 } : file) });
    expect(wrapper.text()).toContain('38.00 KiB');
    await wrapper.findAll('button').find(button => button.text() === '大小').trigger('click');
    expect(wrapper.findAll('tbody tr')[0].text()).toContain('mods');
    wrapper.unmount();
  });
});
