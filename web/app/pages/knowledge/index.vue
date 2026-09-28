<script setup lang="ts">
import type { Article, Topic } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: 'auth',
});

const auth = useAuthStore();
const kb = useKnowledge();
const isStaff = computed(
  () => auth.user?.role === 'admin' || auth.user?.role === 'teacher',
);

const topics = ref<Topic[]>([]);
const articlesByTopic = ref<Record<string, Article[]>>({});
const selectedTopicId = ref('');
const selectedArticle = ref<Article | null>(null);
const loading = ref(true);
const error = ref('');
const editTitle = ref('');
const editBody = ref('');
const newTopicTitle = ref('');
const creating = ref(false);

const selectedTopic = computed(
  () => topics.value.find((t) => t.id === selectedTopicId.value) || null,
);
const articles = computed(
  () => articlesByTopic.value[selectedTopicId.value] || [],
);

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    topics.value = await kb.listTopics();
    if (!selectedTopicId.value && topics.value[0]) {
      selectedTopicId.value = topics.value[0].id;
    }
    if (selectedTopicId.value) {
      await loadArticles(selectedTopicId.value);
    }
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function loadArticles(topicId: string): Promise<void> {
  articlesByTopic.value[topicId] = await kb.listArticles(topicId);
  const list = articlesByTopic.value[topicId] || [];
  if (!selectedArticle.value || selectedArticle.value.topic_id !== topicId) {
    selectedArticle.value = list[0] || null;
    syncEdit();
  }
}

function syncEdit(): void {
  editTitle.value = selectedArticle.value?.title || '';
  editBody.value = selectedArticle.value?.body_md || '';
}

async function selectTopic(id: string): Promise<void> {
  selectedTopicId.value = id;
  await loadArticles(id);
}

function selectArticle(article: Article): void {
  selectedArticle.value = article;
  syncEdit();
}

async function createTopic(): Promise<void> {
  if (!newTopicTitle.value.trim()) {
    return;
  }
  creating.value = true;
  try {
    const topic = await kb.createTopic(newTopicTitle.value.trim());
    newTopicTitle.value = '';
    await load();
    selectedTopicId.value = topic.id;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    creating.value = false;
  }
}

async function createArticle(): Promise<void> {
  if (!selectedTopicId.value) {
    return;
  }
  creating.value = true;
  try {
    const article = await kb.createArticle(selectedTopicId.value, {
      title: 'Новая статья',
      body_md: '',
    });
    await loadArticles(selectedTopicId.value);
    selectArticle(article);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    creating.value = false;
  }
}

async function saveArticle(): Promise<void> {
  if (!selectedArticle.value) {
    return;
  }
  creating.value = true;
  try {
    selectedArticle.value = await kb.updateArticle(selectedArticle.value.id, {
      title: editTitle.value.trim(),
      body_md: editBody.value,
    });
    await loadArticles(selectedTopicId.value);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    creating.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet">
      <header class="sheet__header">
        <div class="sheet__header-row">
          <div>
            <h1 class="page-title">Справочная база</h1>
            <p class="page-sub">
              {{ topics.length }} тем · регламенты и типовые фразы
            </p>
          </div>
          <div v-if="isStaff" class="actions">
            <TbField label="Новая тема">
              <TbInput v-model="newTopicTitle" placeholder="Название темы" />
            </TbField>
            <TbButton :busy="creating" @click="createTopic">Добавить тему</TbButton>
          </div>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else class="kb-layout">
          <aside class="kb-topics">
            <h2 class="sheet__section-title">Разделы</h2>
            <button
              v-for="topic in topics"
              :key="topic.id"
              type="button"
              class="topic"
              :class="{ 'topic--active': topic.id === selectedTopicId }"
              @click="selectTopic(topic.id)"
            >
              {{ topic.title }}
            </button>
            <p v-if="!topics.length" class="curriculum-empty">Тем пока нет</p>
          </aside>
          <aside class="kb-articles">
            <div class="kb-articles__head">
              <h2 class="sheet__section-title">{{ selectedTopic?.title || 'Статьи' }}</h2>
              <TbButton
                v-if="isStaff && selectedTopicId"
                variant="ghost"
                :busy="creating"
                @click="createArticle"
              >
                + статья
              </TbButton>
            </div>
            <button
              v-for="article in articles"
              :key="article.id"
              type="button"
              class="article"
              :class="{ 'article--active': article.id === selectedArticle?.id }"
              @click="selectArticle(article)"
            >
              {{ article.title }}
            </button>
            <p v-if="!articles.length" class="curriculum-empty">Статей нет</p>
          </aside>
          <article class="kb-content">
            <template v-if="selectedArticle">
              <template v-if="isStaff">
                <TbField label="Заголовок">
                  <TbInput v-model="editTitle" />
                </TbField>
                <TbField label="Текст (markdown)">
                  <TbTextarea v-model="editBody" :rows="16" />
                </TbField>
                <TbButton :busy="creating" @click="saveArticle">Сохранить статью</TbButton>
              </template>
              <template v-else>
                <h2>{{ selectedArticle.title }}</h2>
                <pre class="md">{{ selectedArticle.body_md }}</pre>
              </template>
            </template>
            <p v-else class="curriculum-empty">Выберите статью</p>
          </article>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.actions {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.kb-layout {
  display: grid;
  grid-template-columns: 220px 240px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  min-height: 480px;
}

.kb-topics,
.kb-articles,
.kb-content {
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.topic,
.article {
  width: 100%;
  padding: 10px 12px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: 500 13px/1.3 var(--font-sans);
}

.topic--active,
.article--active {
  background: var(--color-secondary);
}

.kb-articles__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.kb-content h2 {
  margin: 0 0 12px;
  font: 700 22px/1.3 var(--font-sans);
}

.md {
  margin: 0;
  white-space: pre-wrap;
  font: 400 14px/1.6 var(--font-sans);
  color: var(--color-text-muted);
}

@media (max-width: 1000px) {
  .kb-layout {
    grid-template-columns: 1fr;
  }
}
</style>
