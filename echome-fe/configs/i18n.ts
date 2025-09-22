import { getRequestConfig } from "next-intl/server";

import { getUserLocale } from "@/services/locale";

export const locales = ["zh", "en"] as const;
export type Locale = (typeof locales)[number];
export const defaultLocale: Locale = "zh";

export default getRequestConfig(async () => {
  const locale = await getUserLocale();
  const messages = (await import(`@/messages/${locale}/index`)).default;

  return {
    locale,
    messages,
  };
});
