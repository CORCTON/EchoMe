"use client";

import { HeroUIProvider } from "@heroui/react";
import { NextIntlClientProvider } from "next-intl";
import { getLocale, getMessages } from "next-intl/server";

export async function Providers({ children }: { children: React.ReactNode }) {
    const messages = await getMessages();
    const locale = await getLocale();
    return (
        <NextIntlClientProvider messages={messages}>
            <HeroUIProvider locale={locale}>
                {children}
            </HeroUIProvider>
        </NextIntlClientProvider>
    );
}
