import type { Metadata } from "next";
import "@/styles/globals.css";
import { getLocale, getMessages } from "next-intl/server";
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
  return (
    <html lang={locale}>
      <body>
        <Providers messages={messages} >{children}</Providers>
      </body>
    </html>
  );
}
