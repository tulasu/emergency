import type { CreateLessonBody, Lesson } from '~/types/curriculum';

export function useLessons() {
  const { $api } = useNuxtApp();

  async function listByModule(moduleId: string): Promise<Lesson[]> {
    return $api<Lesson[]>(`/modules/${moduleId}/lessons`);
  }

  async function pool(q = ''): Promise<Lesson[]> {
    return $api<Lesson[]>('/lessons', {
      query: q ? { q } : undefined,
    });
  }

  async function create(moduleId: string, body: CreateLessonBody): Promise<Lesson> {
    return $api<Lesson>(`/modules/${moduleId}/lessons`, { method: 'POST', body });
  }

  async function archive(lessonId: string): Promise<Lesson> {
    return $api<Lesson>(`/lessons/${lessonId}/archive`, { method: 'POST', body: {} });
  }

  async function copyFromPool(
    moduleId: string,
    lessonId: string,
    position: number,
  ): Promise<Lesson> {
    return $api<Lesson>(`/modules/${moduleId}/lessons/from-pool`, {
      method: 'POST',
      body: { lesson_id: lessonId, position },
    });
  }

  return { listByModule, pool, create, archive, copyFromPool };
}
