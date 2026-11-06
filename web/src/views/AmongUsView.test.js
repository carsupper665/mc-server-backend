import { shallowMount, flushPromises } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import AmongUsView from './AmongUsView.vue';
import api from '../api';

vi.mock('../api', () => ({ default: { get: vi.fn() } }));
vi.mock('../store/activityLog', () => ({ useActivityLogStore: () => ({}) }));
vi.mock('naive-ui', async importOriginal => ({
  ...await importOriginal(),
  useMessage: () => ({})
}));

describe('Among Us room players', () => {
  it('loads message players, retains only IDs and shows the count after loading', async () => {
    api.get.mockResolvedValueOnce({ message: 'ok', games: ['room123'] });
    const wrapper = shallowMount(AmongUsView, {
      global: {
        renderStubDefaultSlot: true,
        stubs: { Button: { template: '<button><slot /></button>' }, Tag: { template: '<span class="player-count"><slot /></span>' }, Card: { template: '<div><slot name="header" /><slot name="header-extra" /><slot /><slot name="footer" /></div>' } }
      }
    });
    try {
      await flushPromises();
      expect(wrapper.find('.player-count').exists()).toBe(false);
      let resolvePlayers;
      api.get.mockReturnValueOnce(new Promise(resolve => { resolvePlayers = resolve; }));
      await wrapper.findAll('button').find(button => button.text() === '查看玩家').trigger('click');
      expect(api.get).toHaveBeenLastCalledWith('/LOL-AmongUs/a/ls-p/room123');
      expect(wrapper.find('.player-count').exists()).toBe(false);
      resolvePlayers({ message: [{ ID: 'player123', Role: 'secret-role', Task: 'secret-task', TaskInf: 'secret-info' }] });
      await flushPromises();
      expect(wrapper.vm.expandedRooms.room123.players).toEqual([{ ID: 'player123' }]);
      expect(wrapper.find('.player-count').text()).toContain('1 玩家');
      expect(wrapper.text()).toContain('player123');
      expect(wrapper.text()).not.toContain('secret-');
    } finally {
      wrapper.unmount();
    }
  });
});
