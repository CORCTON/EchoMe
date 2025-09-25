"use client";

"use client";

import { useCallback, useEffect, useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import Image from "next/image";
import { useLocale, useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";

interface VoiceCharacter {
  id: string;
  name: string;
  image: string;
  description: {
    [key: string]: string;
  };
}

const voiceCharacters: VoiceCharacter[] = [
  {
    id: "sol",
    name: "Sol",
    image:
      "https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=400&h=400&fit=crop&crop=face",
    description: {
      zh: "阳光随性",
      en: "Sunny and casual",
    },
  },
  {
    id: "cove",
    name: "Cove",
    image:
      "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=400&h=400&fit=crop&crop=face",
    description: {
      zh: "沉稳内敛",
      en: "Calm and introverted",
    },
  },
  {
    id: "juniper",
    name: "Juniper",
    image:
      "https://images.unsplash.com/photo-1494790108755-2616b612b786?w=400&h=400&fit=crop&crop=face",
    description: {
      zh: "开放轻松",
      en: "Open and relaxed",
    },
  },
  {
    id: "sage",
    name: "Sage",
    image:
      "https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=400&h=400&fit=crop&crop=face",
    description: {
      zh: "智慧温和",
      en: "Wise and gentle",
    },
  },
  {
    id: "nova",
    name: "Nova",
    image:
      "https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=400&h=400&fit=crop&crop=face",
    description: {
      zh: "活力四射",
      en: "Energetic",
    },
  },
];

export default function Home() {
  const t = useTranslations("home");
  const locale = useLocale();
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isAnimating, setIsAnimating] = useState(false);
  const router = useRouter();

  const currentCharacter = voiceCharacters[currentIndex];

  const nextCharacter = useCallback(() => {
    if (isAnimating) return;
    setIsAnimating(true);
    setCurrentIndex((prev) => (prev + 1) % voiceCharacters.length);
    setTimeout(() => setIsAnimating(false), 300);
  }, [isAnimating]);

  const prevCharacter = useCallback(() => {
    if (isAnimating) return;
    setIsAnimating(true);
    setCurrentIndex((prev) =>
      (prev - 1 + voiceCharacters.length) % voiceCharacters.length
    );
    setTimeout(() => setIsAnimating(false), 300);
  }, [isAnimating]);

  // 自动轮播
  useEffect(() => {
    const interval = setInterval(() => {
      if (!isAnimating) {
        nextCharacter();
      }
    }, 4000);

    return () => clearInterval(interval);
  }, [isAnimating, nextCharacter]);

  return (
    <div className="h-[100dvh] select-none bg-gradient-to-br from-slate-50 to-slate-200 dark:from-slate-900 dark:to-slate-800 flex flex-col items-center justify-center p-6">
      {/* 标题 */}
      <div className="text-center mb-8">
        <h1 className="text-2xl font-bold text-slate-800 dark:text-slate-200 mb-2">
          {t("title")}
        </h1>
        <p className="text-slate-600 dark:text-slate-400">
          {t("description")}
        </p>
      </div>

      {/* 中央圆形角色图像 */}
      <div className="relative mb-12">
        {/* 主要圆形容器 */}
        <div className="relative w-56 h-56 rounded-full overflow-hidden border-4 border-white dark:border-slate-700 shadow-2xl">
          {/* 角色图像 */}
          <Image
            src={currentCharacter.image}
            alt={currentCharacter.name}
            className="absolute inset-0 object-cover"
            width={256}
            height={256}
          />

          {/* 悬浮效果 */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent" />
        </div>

        {/* 导航按钮 */}
        <Button
          variant="outline"
          size="icon"
          onClick={prevCharacter}
          disabled={isAnimating}
          className="absolute left-[-60px] top-1/2 -translate-y-1/2 rounded-full"
        >
          <ChevronLeft size={20} />
        </Button>

        <Button
          variant="outline"
          size="icon"
          onClick={nextCharacter}
          disabled={isAnimating}
          className="absolute right-[-60px] top-1/2 -translate-y-1/2 rounded-full"
        >
          <ChevronRight size={20} />
        </Button>
      </div>

      {/* 底部轮播文字 */}
      <div className="text-center mb-8">
        <div className="relative h-20 overflow-hidden">
          <h2 className="text-xl font-semibold text-slate-800 dark:text-slate-200 mb-2">
            {currentCharacter.name}
          </h2>
          <p className="text-slate-600 dark:text-slate-400">
            {currentCharacter.description[locale]}
          </p>
        </div>

        {/* 指示器 */}
        <div className="flex justify-center space-x-2 mt-6">
          {voiceCharacters.map((character, index) => (
            <Button
              key={character.id}
              variant="ghost"
              onClick={() => {
                if (!isAnimating && index !== currentIndex) {
                  setIsAnimating(true);
                  setCurrentIndex(index);
                  setTimeout(() => setIsAnimating(false), 300);
                }
              }}
              className={`h-2 rounded-full transition-all duration-200 p-0 ${
                index === currentIndex
                  ? "bg-slate-800 dark:bg-slate-200 w-8"
                  : "bg-slate-400 dark:bg-slate-600 w-2"
              }`}
            />
          ))}
        </div>
      </div>

      {/* 操作按钮 */}
      <div className="flex flex-col space-y-4 w-full max-w-[280px]">
        <Button
          size="lg"
          className="w-full rounded-2xl"
          onClick={() => {
            router.push(`/${currentCharacter.id}`);
          }}
        >
          {t("talk_to", { characterName: currentCharacter.name })}
        </Button>
      </div>
    </div>
  );
}
