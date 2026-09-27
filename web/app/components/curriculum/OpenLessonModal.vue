<script setup lang="ts">
import type { Lesson, OpenVariantMode, Variant } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { variantStatusLabel } from '~/utils/curriculum-labels';
import { ruCount } from '~/utils/ru-count';

const props = defineProps<{
  moduleId: string;
  moduleTitle?: string;
  lessons?: Lesson[];
  variants: Variant[];
}>();

const open = defineModel<boolean>('open', { default: false });

const emit = defineEmits<{
  done: [granted: number];
}>();

const attemptsApi = useAttempts();
const { groupOptions, load: loadGroups } = useGroups();

const mode = ref('all');
const lessonId = ref('');
const variantId = ref('');
const selectedGroupIds = ref<string[]>([]);
const userIdsRaw = ref('');
const availableFrom = ref('');
const deadlineAt = ref('');
const busy = ref(false);
const error = ref('');

const modeItems = [
  { id: 'all', label: 'Всем' },
  { id: 'groups', label: 'По группам' },
  { id: 'users', label: 'Ученику' },
];

const lessonOptions = computed(() =>
  (props.lessons || []).map((l) => ({ value: l.id, label: l.title })),
);

const selectedLesson = computed(
  () => (props.lessons || []).find((l) => l.id === lessonId.value) || null,
);

const variantsForLesson = computed(() => {
  if (!lessonId.value) {
    return props.variants;
  }
  const filtered = props.variants.filter((v) => v.lesson_id === lessonId.value);
  return filtered.length ? filtered : props.variants;
});

const recipientCount = computed(() => {
  if (mode.value === 'groups') {
    return selectedGroupIds.value.reduce((sum, id) => {
      const opt = groupOptions.value.find((g) => g.value === id);
      const match = opt?.label?.match(/(\d+)\s*$/);
      return sum + (match ? Number(match[1]) : 1);
    }, 0);
  }
  if (mode.value === 'users') {
    return userIdsRaw.value
      .split(/[\s,;]+/)
      .map((s) => s.trim())
      .filter(Boolean).length;
  }
  return groupOptions.value.reduce((sum, opt) => {
    const match = opt.label?.match(/\((\d+)\)/);
    return sum + (match ? Number(match[1]) : 1);
  }, groupOptions.value.length ? 0 : 0) || groupOptions.value.length;
});

const primaryCta = computed(() => {
  const n = recipientCount.value;
  return n > 0 ? `Открыть для ${n}` : 'Открыть занятие';
});

const footerMeta = computed(() => {
  const n = recipientCount.value;
  const attemptLine = n > 0 ? `${n} попыток · № 1` : 'Попытки · № 1';
  if (availableFrom.value) {
    const d = new Date(availableFrom.value);
    const formatted = Number.isNaN(d.getTime())
      ? availableFrom.value
      : d.toLocaleString('ru-RU', {
          day: '2-digit',
          month: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
        });
    return { attemptLine, schedule: `«Можно приступить» с ${formatted}` };
  }
  return { attemptLine, schedule: '' };
});

const modeHint = computed(() => {
  if (mode.value === 'all') {
    return 'Одно окно и один вариант для всех, у кого занятие ещё не открыто.';
  }
  if (mode.value === 'groups') {
    return 'Выберите группы — попытки получат только их участники.';
  }
  return 'Укажите учеников по ID — для точечной выдачи.';
});

watch(
  () => open.value,
  async (isOpen) => {
    if (!isOpen) {
      return;
    }
    error.value = '';
    if (!lessonId.value && props.lessons?.[0]) {
      lessonId.value = props.lessons[0].id;
    }
    syncVariant();
    try {
      await loadGroups();
    } catch {
      // optional
    }
  },
);

watch(lessonId, () => {
  syncVariant();
});

function syncVariant(): void {
  const list = variantsForLesson.value;
  if (!list.find((v) => v.id === variantId.value)) {
    const primary = list.find((v) => v.is_primary);
    variantId.value = primary?.id || list[0]?.id || '';
  }
}

function toggleGroup(id: string): void {
  if (selectedGroupIds.value.includes(id)) {
    selectedGroupIds.value = selectedGroupIds.value.filter((g) => g !== id);
  } else {
    selectedGroupIds.value = [...selectedGroupIds.value, id];
  }
}

function variantSub(variant: Variant): string {
  const status = variant.is_primary
    ? 'основной'
    : variantStatusLabel(variant.status).toLowerCase();
  const tickets = variant.ticket_count ?? 0;
  return `${status} · ${ruCount(tickets, 'задание', 'задания', 'заданий')}`;
}

async function submit(): Promise<void> {
  if (!variantId.value) {
    error.value = 'Выберите вариант';
    return;
  }
  busy.value = true;
  error.value = '';
  try {
    const body = {
      mode: mode.value as OpenVariantMode,
      module_id: mode.value === 'all' ? props.moduleId : undefined,
      group_ids: mode.value === 'groups' ? selectedGroupIds.value : [],
      user_ids:
        mode.value === 'users'
          ? userIdsRaw.value
              .split(/[\s,;]+/)
              .map((s) => s.trim())
              .filter(Boolean)
          : [],
      available_from: availableFrom.value || undefined,
      deadline_at: deadlineAt.value || undefined,
    };
    const res = await attemptsApi.openVariant(variantId.value, body);
    open.value = false;
    emit('done', res.granted);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <TbModal
    v-model:open="open"
    title="Открыть занятие"
    :subtitle="moduleTitle ? `Модуль «${moduleTitle}»` : undefined"
  >
    <div class="open-modal">
      <div v-if="lessonOptions.length" class="lesson-select">
        <TbIcon name="book" class="lesson-select__icon" />
        <div class="lesson-select__text">
          <span class="lesson-select__label">Занятие</span>
          <strong>{{ selectedLesson?.title || 'Выберите занятие' }}</strong>
        </div>
        <span class="lesson-select__state">ещё не открыто ни для кого</span>
        <select v-model="lessonId" class="lesson-select__native" aria-label="Занятие">
          <option v-for="opt in lessonOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <TbIcon name="chevron-down" class="lesson-select__chevron" />
      </div>

      <div class="mode-block">
        <TbSegmented v-model="mode" :items="modeItems" />
        <p class="mode-hint">{{ modeHint }}</p>
      </div>

      <div v-if="mode === 'all'" class="who-box">
        <div class="who-box__head">
          <span>Получат занятие</span>
          <strong>{{
            recipientCount
              ? ruCount(recipientCount, 'ученик', 'ученика', 'учеников')
              : '—'
          }}</strong>
        </div>
        <div class="who-chips">
          <TbChip v-for="opt in groupOptions" :key="opt.value" disabled>
            {{ opt.label }}
          </TbChip>
        </div>
      </div>

      <div v-else-if="mode === 'groups'" class="group-list">
        <button
          v-for="opt in groupOptions"
          :key="opt.value"
          type="button"
          class="group-row"
          :class="{ 'group-row--active': selectedGroupIds.includes(opt.value) }"
          @click="toggleGroup(opt.value)"
        >
          <span>{{ opt.label }}</span>
          <TbIcon v-if="selectedGroupIds.includes(opt.value)" name="check" />
        </button>
      </div>

      <TbField v-else label="ID пользователей">
        <TbTextarea
          v-model="userIdsRaw"
          placeholder="uuid через запятую или с новой строки"
          :rows="3"
        />
      </TbField>

      <div class="dates">
        <TbField label="Можно приступить с">
          <TbDateTimePopover v-model="availableFrom" />
        </TbField>
        <TbField label="Дедлайн">
          <TbDateTimePopover v-model="deadlineAt" />
        </TbField>
      </div>

      <div class="variant-pick">
        <p class="variant-pick__title">Основной вариант</p>
        <div class="variant-pick__options">
          <button
            v-for="variant in variantsForLesson"
            :key="variant.id"
            type="button"
            class="variant-radio"
            :class="{ 'variant-radio--active': variantId === variant.id }"
            @click="variantId = variant.id"
          >
            <span class="variant-radio__dot"></span>
            <div class="variant-radio__text">
              <strong>{{ variant.title }}</strong>
              <span>{{ variantSub(variant) }}</span>
            </div>
          </button>
        </div>
      </div>

      <p v-if="error" class="curriculum-error">{{ error }}</p>
    </div>
    <template #footer>
      <div class="footer-meta">
        <strong>{{ footerMeta.attemptLine }}</strong>
        <span v-if="footerMeta.schedule">{{ footerMeta.schedule }}</span>
      </div>
      <TbButton variant="secondary" @click="open = false">Отмена</TbButton>
      <TbButton :busy="busy" @click="submit">{{ primaryCta }}</TbButton>
    </template>
  </TbModal>
</template>

<style scoped>
.open-modal {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.lesson-select {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  padding: 0 16px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
}

.lesson-select__icon {
  color: var(--color-primary);
  flex-shrink: 0;
}

.lesson-select__text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.lesson-select__label {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.lesson-select__text strong {
  font: var(--font-body);
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lesson-select__state {
  color: var(--color-text-subtle);
  font: var(--font-cap);
  flex-shrink: 0;
}

.lesson-select__chevron {
  color: var(--color-text-muted);
  flex-shrink: 0;
  pointer-events: none;
}

.lesson-select__native {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
  width: 100%;
  height: 100%;
}

.mode-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mode-hint {
  margin: 0;
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.who-box {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.who-box__head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
  font: var(--font-body);
  color: var(--color-text);
}

.who-box__head strong {
  color: var(--color-primary);
  font: var(--font-body);
}

.who-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.group-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.group-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.group-row--active {
  border-color: var(--color-primary);
  background: var(--color-secondary);
}

.dates {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.variant-pick {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.variant-pick__title {
  margin: 0;
  font: var(--font-body);
  color: var(--color-text);
}

.variant-pick__options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.variant-radio {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 200px;
  padding: 12px 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  text-align: left;
  cursor: pointer;
  font: inherit;
}

.variant-radio--active {
  border-color: var(--color-primary);
  background: var(--color-secondary);
}

.variant-radio__dot {
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-border-strong);
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--color-surface);
  box-sizing: border-box;
}

.variant-radio--active .variant-radio__dot {
  border-color: var(--color-primary);
  box-shadow: inset 0 0 0 4px var(--color-primary);
}

.variant-radio__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.variant-radio__text strong {
  font: var(--font-body);
  color: var(--color-text);
}

.variant-radio--active .variant-radio__text strong {
  color: var(--color-primary);
}

.variant-radio__text span {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.footer-meta {
  margin-right: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.footer-meta strong {
  font: var(--font-body);
  color: var(--color-text);
}

.footer-meta span {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

@media (max-width: 640px) {
  .dates {
    grid-template-columns: 1fr;
  }

  .variant-pick__options {
    flex-direction: column;
  }

  .lesson-select__state {
    display: none;
  }
}
</style>
