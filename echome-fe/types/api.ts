export type APIError = {
  code?: string;
  message?: string;
  details?: string;
};

export type APIResponse<T = unknown> = {
  success: boolean;
  data?: T;
  error?: APIError | null;
};

/**
 * 表示 parseApiResponse 的返回形态
 */
export type ParseApiResponseResult<T = unknown> =
  | { ok: true; value: APIResponse<T> }
  | { ok: false; reason: 'invalid-json' | 'not-api-response'; value?: unknown };
