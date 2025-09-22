import type { Metadata } from "next";
import "@/styles/globals.css";
import { getLocale } from "next-intl/server";

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
        {children}
      </body>
    </html>
  );
}
