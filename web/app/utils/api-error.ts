export interface HumaError {
  status?: number;
  title?: string;
  detail?: string;
}

export type ApiErrorCode =
  | 'invalid_credentials'
  | 'unauthorized'
  | 'user_blocked'
  | 'forbidden'
  | 'conflict'
  | 'invalid_input'
  | 'not_found'
  | 'failed';

const MESSAGES: Record<ApiErrorCode, string> = {
  invalid_credentials: 'Неверный логин или пароль',
  unauthorized: 'Нужно войти заново',
  user_blocked: 'Пользователь заблокирован',
  forbidden: 'Недостаточно прав',
  conflict: 'Такой логин уже занят',
  invalid_input: 'Проверьте введённые данные',
  not_found: 'Не найдено',
  failed: 'Не удалось выполнить запрос',
};

const KNOWN = new Set<string>(Object.keys(MESSAGES));

export function readErrorCode(error: unknown): ApiErrorCode {
  const err = error as {
    status?: number;
    statusCode?: number;
    data?: HumaError;
    error?: HumaError;
    response?: { status?: number; _data?: HumaError };
  };
  const detail = err?.data?.detail ?? err?.error?.detail ?? err?.response?._data?.detail;
  if (detail && KNOWN.has(detail)) {
    return detail as ApiErrorCode;
  }
  const status = err?.statusCode ?? err?.status ?? err?.data?.status ?? err?.response?.status;
  if (status === 401) {
    return 'unauthorized';
  }
  if (status === 403) {
    return 'forbidden';
  }
  if (status === 409) {
    return 'conflict';
  }
  if (status === 400) {
    return 'invalid_input';
  }
  if (status === 404) {
    return 'not_found';
  }
  return 'failed';
}

export function errorCodeMessage(code: string | undefined, fallback = MESSAGES.failed): string {
  if (code && KNOWN.has(code)) {
    return MESSAGES[code as ApiErrorCode];
  }
  return fallback;
}

export function apiErrorMessage(error: unknown, fallback = MESSAGES.failed): string {
  return errorCodeMessage(readErrorCode(error), fallback);
}
