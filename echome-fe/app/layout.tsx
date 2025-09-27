import type { Metadata } from "next";
import "@/styles/globals.css";
import { getLocale, getMessages, getTimeZone } from "next-intl/server";
import { Providers } from "./provider";

export const metadata: Metadata = {
  title: "EchoMe",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const locale = await getLocale();
  const messages = await getMessages();
  const timeZone = await getTimeZone();
  return (
    <html lang={locale}>
      <body>
        <Providers messages={messages} locale={locale} timeZone={timeZone}>
          {children}
        </Providers>
      </body>
    </html>
  );
}
