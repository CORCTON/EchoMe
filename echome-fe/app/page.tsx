"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { CharacterCarousel } from "@/components/ui/character-carousel";
import { ModelSettingsDrawer, type ModelSettings } from "@/components/model-settings-drawer";
import { useRouter } from "next/navigation";
import { voiceCharacters, type VoiceCharacter } from "@/lib/characters";

export default function Home() {
  const t = useTranslations("home");
  const router = useRouter();
  const [selectedCharacter, setSelectedCharacter] = useState<VoiceCharacter>(
    voiceCharacters[0],
  );
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  const handleCharacterSelect = (character: VoiceCharacter) => {
    setSelectedCharacter(character);
  };

  const handleTalkToCharacter = () => {
    setIsSettingsOpen(true);
  };

  const handleSettingsReady = (settings: ModelSettings) => {
    // 这里可以保存设置到状态管理或localStorage
    console.log('Model settings:', settings);
    
    // 导航到聊天页面
    router.push(`/${selectedCharacter.id}`);
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
          <CharacterCarousel
            characters={voiceCharacters}
            selectedCharacter={selectedCharacter}
            onCharacterSelect={handleCharacterSelect}
          />
        </div>

        {/* 操作按钮 */}
        <div className="flex flex-col space-y-4 w-full max-w-[280px]">
          <Button
            size="lg"
            className="w-full rounded-2xl cursor-pointer"
            onClick={handleTalkToCharacter}
          >
            {t("talk_to", { characterName: selectedCharacter.name })}
          </Button>
        </div>
      </div>

      {/* 模型设置抽屉 */}
      <ModelSettingsDrawer
        open={isSettingsOpen}
        onOpenChange={setIsSettingsOpen}
        character={selectedCharacter}
        onReady={handleSettingsReady}
      />
    </>
  );
}
