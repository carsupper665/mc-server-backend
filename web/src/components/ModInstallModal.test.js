import { shallowMount, flushPromises } from '@vue/test-utils';
import { expect, it, vi } from 'vitest';
import { NSelect } from 'naive-ui';
import ModInstallModal from './ModInstallModal.vue';
import { getProjectVersions } from '../api/modrinth';

vi.mock('../api/modrinth', () => ({ getProjectVersions: vi.fn().mockResolvedValue([]), formatDownloads: String }));

it('uses an empty selection for the placeholder and resets it when reopened', async () => {
  const wrapper = shallowMount(ModInstallModal, {
    props: {
      show: true,
      mod: { id: 'mod', categories: ['fabric'], versions: ['1.21'] },
      servers: [{ server_id: 'server', mod_loader: 'Fabric', mc_version: '1.21' }],
    },
    global: { renderStubDefaultSlot: true },
  });
  try {
    const select = wrapper.findAllComponents(NSelect)[0];
    expect(select.props('placeholder')).toBe('選擇要安裝的伺服器');
    expect(select.props('value')).toBeNull();
    select.vm.$emit('update:value', 'server');
    await flushPromises();
    expect(getProjectVersions).toHaveBeenCalledWith('mod', { loaders: ['fabric'], game_versions: ['1.21'] });
    await wrapper.setProps({ show: false });
    getProjectVersions.mockClear();
    await wrapper.setProps({ show: true });
    await flushPromises();
    expect(select.props('value')).toBeNull();
    expect(getProjectVersions).not.toHaveBeenCalled();
    wrapper.vm.$.setupState.handleInstall();
    expect(wrapper.emitted('install')).toBeUndefined();
  } finally {
    wrapper.unmount();
  }
});
