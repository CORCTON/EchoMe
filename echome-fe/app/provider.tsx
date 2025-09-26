import { NextIntlClientProvider } from "next-intl";

export async function Providers({ children,messages }: { children: React.ReactNode ,messages: Record<string, string> }) {
  return (
    <NextIntlClientProvider messages={messages}>
      {children}
    </NextIntlClientProvider>
  );
}
