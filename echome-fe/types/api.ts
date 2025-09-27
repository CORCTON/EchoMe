export type APIError = {
  code?: string;
  message?: string;
  details?: string;
};

export interface APIResponse<T> {
  success: boolean;
  data: T;
  error?: APIError;
}
