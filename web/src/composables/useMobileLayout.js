import { ref, onMounted, onBeforeUnmount } from 'vue';

export function useMobileLayout() {
  const query = window.matchMedia('(max-width: 767px)');
  const isMobile = ref(query.matches);
  const update = (event) => { isMobile.value = event.matches; };

  onMounted(() => {
    isMobile.value = query.matches;
    query.addEventListener('change', update);
  });
  onBeforeUnmount(() => query.removeEventListener('change', update));

  return isMobile;
}
