<script setup lang="ts">
import type { Variant } from '~/types/curriculum';
import { variantStatusTone } from '~/utils/curriculum-labels';
import { ruCount } from '~/utils/ru-count';

const props = defineProps<{
  variant: Variant;
  ticketCount?: number;
  attemptCount?: number;
  avgSuccess?: number | null;
  durationMinutes?: number | null;
  topics?: string[];
  missingTopics?: string[];
}>();

const emit = defineEmits<{
  open: [];
  makePrimary: [];
}>();

const statusLabel = computed(() => {
  if (props.variant.is_primary) {
    return '★ основной';
  }
  if (props.variant.status === 'approved') {
    return 'утверждён';
  }
  return 'черновик';
});

const statusTone = computed(() =>
  props.variant.is_primary ? 'info' : variantStatusTone(props.variant.status),
);

const normLabel = computed(() => {
  if (props.durationMinutes != null && props.durationMinutes > 0) {
    return `${props.durationMinutes} мин`;
  }
  const tickets = props.ticketCount ?? 0;
  return tickets > 0 ? `${tickets * 3} мин` : '—';
});

const avgLabel = computed(() =>
  props.avgSuccess != null ? `${Math.round(props.avgSuccess)}%` : '—',
);

const avgTone = computed(() =>
  props.avgSuccess != null && props.avgSuccess >= 70 ? 'good' : 'plain',
);

const metaLabel = computed(() => {
  const attempts = props.attemptCount ?? 0;
  const attemptsPart =
    attempts > 0 ? ruCount(attempts, 'попытка', 'попытки', 'попыток') : 'не использовался';
  const date = props.variant.created_at
    ? new Date(props.variant.created_at).toLocaleDateString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
      })
    : '';
  return date ? `${attemptsPart} · изменён ${date}` : attemptsPart;
});

const showMakePrimary = computed(
  () => !props.variant.is_primary && props.variant.status === 'approved',
);

const primaryCta = computed(() =>
  props.variant.status === 'draft' ? 'Продолжить сборку' : 'Открыть редактор',
);
</script>

<template>
  <article
    class="variant-overview-card"
    :class="{ 'variant-overview-card--primary': variant.is_primary }"
  >
    <div class="variant-overview-card__head">
      <div class="variant-overview-card__title-row">
        <h3>{{ variant.title }}</h3>
        <TbBadge :tone="statusTone">{{ statusLabel }}</TbBadge>
      </div>
    </div>

    <div class="variant-overview-card__stats">
      <div class="variant-overview-card__stat">
        <strong>{{ ticketCount ?? 0 }}</strong>
        <span>заданий</span>
      </div>
      <div class="variant-overview-card__stat">
        <strong>{{ normLabel }}</strong>
        <span>норматив</span>
      </div>
      <div class="variant-overview-card__stat">
        <strong :class="{ 'is-good': avgTone === 'good' }">{{ avgLabel }}</strong>
        <span>средний результат</span>
      </div>
    </div>

    <div
      v-if="(topics && topics.length) || (missingTopics && missingTopics.length)"
      class="variant-overview-card__topics"
    >
      <span class="variant-overview-card__topics-label">Темы:</span>
      <TbChip v-for="topic in topics || []" :key="topic" disabled>{{ topic }}</TbChip>
      <TbChip
        v-for="topic in missingTopics || []"
        :key="`missing-${topic}`"
        disabled
        class="variant-overview-card__missing"
      >
        нет «{{ topic }}»
      </TbChip>
    </div>

    <p class="variant-overview-card__meta">{{ metaLabel }}</p>

    <div class="variant-overview-card__actions">
      <TbButton
        :variant="variant.status === 'draft' ? 'primary' : 'secondary'"
        width="block"
        @click="emit('open')"
      >
        {{ primaryCta }}
      </TbButton>
      <TbButton
        v-if="showMakePrimary"
        variant="secondary"
        width="block"
        @click="emit('makePrimary')"
      >
        Сделать основным
      </TbButton>
    </div>
  </article>
</template>

<style scoped>
.variant-overview-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  min-height: 260px;
  padding: 20px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  box-shadow: var(--shadow-card);
  box-sizing: border-box;
  overflow: hidden;
}

.variant-overview-card--primary {
  border-color: var(--color-primary);
  border-width: 1.5px;
}

.variant-overview-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.variant-overview-card__title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.variant-overview-card__head h3 {
  margin: 0;
  font: var(--font-block);
  color: var(--color-text);
}

.variant-overview-card__stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.variant-overview-card__stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  background: #f3f3f3;
  min-width: 0;
}

.variant-overview-card__stat strong {
  font: var(--font-block);
  color: var(--color-text);
}

.variant-overview-card__stat strong.is-good {
  color: var(--color-success);
}

.variant-overview-card__stat span {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.variant-overview-card__topics {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.variant-overview-card__topics-label {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.variant-overview-card__missing {
  background: var(--color-warning-soft) !important;
  color: var(--color-warning) !important;
}

.variant-overview-card__meta {
  margin: 0;
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.variant-overview-card__actions {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  min-width: 0;
}

.variant-overview-card__actions :deep(.tb-btn) {
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}
</style>
