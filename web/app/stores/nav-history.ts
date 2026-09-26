export const useNavHistoryStore = defineStore('nav-history', () => {
  const stack = ref<string[]>([]);
  const canGoBack = computed(() => stack.value.length > 1);

  let trigger: 'imperative' | 'popstate' = 'imperative';
  let replaceUrl = false;

  function markPop(): void {
    trigger = 'popstate';
  }

  function markReplace(): void {
    replaceUrl = true;
  }

  function onEnd(url: string, skip: boolean): void {
    if (skip) {
      stack.value = [];
      trigger = 'imperative';
      replaceUrl = false;
      return;
    }

    const path = normalize(url);
    if (trigger === 'popstate') {
      const idx = stack.value.lastIndexOf(path);
      if (idx >= 0) {
        stack.value = stack.value.slice(0, idx + 1);
      } else {
        stack.value.push(path);
      }
    } else if (replaceUrl && stack.value.length > 0) {
      stack.value[stack.value.length - 1] = path;
    } else if (stack.value[stack.value.length - 1] !== path) {
      stack.value.push(path);
    }

    trigger = 'imperative';
    replaceUrl = false;
  }

  function back(): void {
    if (!canGoBack.value) {
      return;
    }
    window.history.back();
  }

  return { canGoBack, markPop, markReplace, onEnd, back };
});

function normalize(url: string): string {
  const path = url.split(/[?#]/)[0] ?? url;
  return path === '' ? '/' : path;
}
