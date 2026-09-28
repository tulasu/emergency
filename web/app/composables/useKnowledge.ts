import type { Article, Topic } from '~/types/curriculum';

export function useKnowledge() {
  const { $api } = useNuxtApp();

  async function listTopics(): Promise<Topic[]> {
    return $api<Topic[]>('/topics');
  }

  async function createTopic(title: string): Promise<Topic> {
    return $api<Topic>('/topics', { method: 'POST', body: { title } });
  }

  async function updateTopic(topicId: string, title: string): Promise<Topic> {
    return $api<Topic>(`/topics/${topicId}`, { method: 'PATCH', body: { title } });
  }

  async function removeTopic(topicId: string): Promise<void> {
    await $api(`/topics/${topicId}`, { method: 'DELETE' });
  }

  async function listArticles(topicId: string): Promise<Article[]> {
    return $api<Article[]>(`/topics/${topicId}/articles`);
  }

  async function getArticle(articleId: string): Promise<Article> {
    return $api<Article>(`/articles/${articleId}`);
  }

  async function createArticle(
    topicId: string,
    body: { title: string; body_md: string },
  ): Promise<Article> {
    return $api<Article>(`/topics/${topicId}/articles`, { method: 'POST', body });
  }

  async function updateArticle(
    articleId: string,
    body: { title: string; body_md: string },
  ): Promise<Article> {
    return $api<Article>(`/articles/${articleId}`, { method: 'PATCH', body });
  }

  async function removeArticle(articleId: string): Promise<void> {
    await $api(`/articles/${articleId}`, { method: 'DELETE' });
  }

  return {
    listTopics,
    createTopic,
    updateTopic,
    removeTopic,
    listArticles,
    getArticle,
    createArticle,
    updateArticle,
    removeArticle,
  };
}
