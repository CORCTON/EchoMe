"use client";

import { useState, useEffect } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { CharacterCarousel } from "@/components/character-carousel";
import {
  ModelSettingsDrawer,
  type ModelSettings,
} from "@/components/model-settings-drawer";
import { useRouter } from "next/navigation";
import { useCharacterStore } from "@/store/character";
import { useFileStore } from "@/store/file";
import type { Character } from "@/types/character";

interface HomePageProps {
  initialCharacters: Character[];
}

export default function HomePage({ initialCharacters }: HomePageProps) {
  const t = useTranslations("home");
  const router = useRouter();
  const {
    characters,
    currentCharacter,
    setCharacters,
    setCurrentCharacter,
    updateModelSettings,
  } = useCharacterStore();
  const { files } = useFileStore();

  useEffect(() => {
    setCharacters(initialCharacters);
  }, [initialCharacters, setCharacters]);

  useEffect(() => {
    if (!currentCharacter && characters.length > 0) {
      setCurrentCharacter(characters[0].id);
    }
  }, [currentCharacter, characters, setCurrentCharacter]);

  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  const handleCharacterSelect = (character: Character) => {
    setCurrentCharacter(character.id);
  };

  const handleTalkToCharacter = () => {
    setIsSettingsOpen(true);
  };

  const handleSettingsReady = (settings: Omit<ModelSettings, "fileUrls">) => {
    const finalSettings = { ...settings, fileUrls: files.map((f) => f.url) };
    updateModelSettings(finalSettings);
    if (currentCharacter) {
      router.push(`/${currentCharacter.id}`);
    }
  };

  return (
    <>
      <div 
        vaul-drawer-wrapper="" 
        className="h-[100dvh] select-none bg-gradient-to-br from-slate-50 to-slate-200 dark:from-slate-900 dark:to-slate-800 flex flex-col items-center justify-center p-6 overflow-hidden"
      >
        {/* 标题 */}
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-slate-800 dark:text-slate-200 mb-2">
            {t("title")}
          </h1>
          <p className="text-slate-600 dark:text-slate-400">{t("description")}</p>
        </div>

        {/* 轮播组件 */}
        <div className="mb-8">
          {currentCharacter && (
            <CharacterCarousel
              characters={characters}
              selectedCharacter={currentCharacter}
              onCharacterSelect={handleCharacterSelect}
            />
          )}
        </div>

        {/* 操作按钮 */}
        <div className="flex flex-col space-y-4 w-full max-w-[280px]">
          {currentCharacter && (
            <Button
              size="lg"
              className="w-full rounded-2xl cursor-pointer"
              onClick={handleTalkToCharacter}
            >
              {t("talk_to", { characterName: currentCharacter.name })}
            </Button>
          )}
        </div>
      </div>

      {/* 模型设置抽屉 */}
      {currentCharacter && (
        <ModelSettingsDrawer
          open={isSettingsOpen}
          onOpenChange={setIsSettingsOpen}
          character={currentCharacter}
          onReady={handleSettingsReady}
        />
      )}
    </>
  );
}
