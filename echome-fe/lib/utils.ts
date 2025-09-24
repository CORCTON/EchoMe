import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import type { APIResponse, ParseApiResponseResult } from "@/types/api"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * 尝试解析任意字符串为 APIResponse<T>。
 * - 如果解析失败，返回 { ok: false, reason: 'invalid-json' }
 * - 如果解析成功但不含 success 字段，返回 { ok: false, reason: 'not-api-response', value }
 * - 否则返回 { ok: true, value: parsed }
 */
export function parseApiResponse<T = unknown>(raw: string): ParseApiResponseResult<T> {
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === 'object') {
      const rec = parsed as Record<string, unknown>;
      if ('success' in rec && typeof rec['success'] === 'boolean') {
        return { ok: true, value: parsed as APIResponse<T> };
      }
    }
    return { ok: false, reason: 'not-api-response', value: parsed };
  } catch (_e) {
    return { ok: false, reason: 'invalid-json' };
  }
}
