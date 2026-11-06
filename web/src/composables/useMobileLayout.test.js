/** @vitest-environment jsdom */
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { mount } from '@vue/test-utils';
import { useMobileLayout } from './useMobileLayout';

afterEach(() => vi.unstubAllGlobals());

describe('useMobileLayout', () => {
  it.each([320, 360, 390, 768, 1280])('selects one layout at %i px and updates on changes', async (width) => {
    let change;
    const removeEventListener = vi.fn();
    const matchMedia = vi.fn(() => ({
      matches: width < 768,
      addEventListener: vi.fn((event, listener) => { change = listener; }),
      removeEventListener,
    }));
    vi.stubGlobal('matchMedia', matchMedia);
    const wrapper = mount(defineComponent({
      setup() {
        const mobile = useMobileLayout();
        return () => mobile.value ? h('div', { class: 'cards' }) : h('table');
      },
    }));
    expect(matchMedia).toHaveBeenCalledWith('(max-width: 767px)');
    expect(wrapper.find('.cards').exists()).toBe(width < 768);
    expect(wrapper.find('table').exists()).toBe(width >= 768);
    change({ matches: width >= 768 });
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.cards').exists()).toBe(width >= 768);
    expect(wrapper.find('table').exists()).toBe(width < 768);
    wrapper.unmount();
    expect(removeEventListener).toHaveBeenCalledWith('change', change);
  });
});
