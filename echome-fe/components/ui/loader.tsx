"use client";
import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";

export function Loader() {
  const t = useTranslations("layout");
  return (
    <div className="flex h-full w-full items-center justify-center gap-2">
      <Loader2 className="animate-spin" />
      <p>{t("loading")}</p>
    </div>
  );
}
