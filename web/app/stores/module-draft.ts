export interface ModuleDraftLesson {
  key: string;
  title: string;
  sourceLessonId?: string;
}

export const useModuleDraftStore = defineStore('module-draft', () => {
  const step = ref(0);
  const title = ref('');
  const description = ref('');
  const successThreshold = ref(70);
  const lessons = ref<ModuleDraftLesson[]>([]);
  const assignGroupId = ref('');

  function reset(): void {
    step.value = 0;
    title.value = '';
    description.value = '';
    successThreshold.value = 70;
    lessons.value = [];
    assignGroupId.value = '';
  }

  function addEmptyLesson(): void {
    lessons.value.push({
      key: `new-${Date.now()}-${lessons.value.length}`,
      title: `Занятие ${lessons.value.length + 1}`,
    });
  }

  function addFromPool(lessonId: string, lessonTitle: string): void {
    lessons.value.push({
      key: `pool-${lessonId}-${Date.now()}`,
      title: lessonTitle,
      sourceLessonId: lessonId,
    });
  }

  function removeLesson(key: string): void {
    lessons.value = lessons.value.filter((item) => item.key !== key);
  }

  return {
    step,
    title,
    description,
    successThreshold,
    lessons,
    assignGroupId,
    reset,
    addEmptyLesson,
    addFromPool,
    removeLesson,
  };
});
