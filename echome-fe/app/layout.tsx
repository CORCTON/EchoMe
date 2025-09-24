import type { Metadata } from "next";
import "@/styles/globals.css";
import { getLocale } from "next-intl/server";
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
  return (
    <html lang={locale}>
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
