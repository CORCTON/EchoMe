"use client";

import { useRef, useState } from "react";
import Image from "next/image";
import { Button } from "@/components/ui/button";
import { ChevronLeft, ChevronRight, Play, Pause } from "lucide-react";
import type { Character } from "@/types/character";

interface CharacterCarouselProps {
  characters: Character[];
  onCharacterSelect: (character: Character) => void;
  selectedCharacter: Character;
}

export function CharacterCarousel({
  characters,
  onCharacterSelect,
  selectedCharacter,
}: CharacterCarouselProps) {
  const [isPlaying, setIsPlaying] = useState(false);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const currentIndex = characters.findIndex(
    (char) => char.id === selectedCharacter.id,
  );
  const voiceSrc = (selectedCharacter.audio_example || "").trim();
  const hasValidVoice = voiceSrc && voiceSrc !== "null";

  const goToPrevious = () => {
    const prevIndex =
      currentIndex > 0 ? currentIndex - 1 : characters.length - 1;
    onCharacterSelect(characters[prevIndex]);
  };

  const goToNext = () => {
    const nextIndex =
      currentIndex < characters.length - 1 ? currentIndex + 1 : 0;
    onCharacterSelect(characters[nextIndex]);
  };

  return (
    <div className="relative w-full max-w-sm mx-auto">
      <div className="flex flex-col items-center justify-center h-[50vh]">
        {/* 角色图像 和 导航按钮容器 */}
        <div className="relative mb-6 flex items-center justify-center space-x-4">
          {/* 左侧按钮 */}
          {characters.length > 1 && (
            <Button
              variant="outline"
              size="icon"
              onClick={goToPrevious}
              className="rounded-full bg-white/90 dark:bg-slate-800/90 backdrop-blur-sm hover:bg-white dark:hover:bg-slate-800 transition-all duration-200 shadow-lg"
              aria-label="上一位角色"
            >
              <ChevronLeft size={20} />
            </Button>
          )}

          <div className="relative w-48 h-48 rounded-full overflow-hidden border-4 border-white dark:border-slate-700 shadow-2xl">
            {selectedCharacter.avatar ? (
              <Image
                src={selectedCharacter.avatar}
                alt={selectedCharacter.name}
                className="w-full h-full object-cover"
                width={256}
                height={256}
                priority
              />
            ) : (
              <div className="w-full h-full bg-slate-200 dark:bg-slate-700" />
            )}
            <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent" />
          </div>

          {/* 语音播放按钮 */}
          {hasValidVoice && (
            <div className="absolute inset-0 flex items-center justify-center">
              <button
                type="button"
                onClick={() => {
                  if (!audioRef.current) return;

                  if (isPlaying) {
                    audioRef.current.pause();
                  } else {
                    audioRef.current.src = voiceSrc;
                    audioRef.current.play();
                  }
                }}
                className="p-3 bg-black/50 rounded-full text-white hover:bg-black/70 transition-all"
                aria-label={`播放 ${selectedCharacter.name} 的语音`}
              >
                {isPlaying ? <Pause size={24} /> : <Play size={24} />}
              </button>
            </div>
          )}

          {/* 右侧按钮 */}
          {characters.length > 1 && (
            <Button
              variant="outline"
              size="icon"
              onClick={goToNext}
              className="rounded-full bg-white/90 dark:bg-slate-800/90 backdrop-blur-sm hover:bg-white dark:hover:bg-slate-800 transition-all duration-200 shadow-lg"
              aria-label="下一位角色"
            >
              <ChevronRight size={20} />
            </Button>
          )}
        </div>

        {/* 角色信息 */}
        <div className="text-center">
          <h2 className="text-2xl font-semibold mb-2 text-slate-800 dark:text-slate-200">
            {selectedCharacter.name}
          </h2>
          <p className="text-sm text-slate-600 dark:text-slate-400 max-w-xs">
            {selectedCharacter.description}
          </p>
        </div>
      </div>

      {/* (已移除) 底部绝对定位导航按钮：已改为头像两侧的按钮 */}

      {/* 指示器 */}
      {characters.length > 1 && (
        <div className="flex justify-center space-x-2 mt-6">
          {characters.map((character) => (
            <button
              key={character.id}
              type="button"
              onClick={() => onCharacterSelect(character)}
              className={`h-2 rounded-full transition-all duration-300 hover:scale-110 ${
                character.id === selectedCharacter.id
                  ? "bg-slate-800 dark:bg-slate-200 w-8"
                  : "bg-slate-400 dark:bg-slate-600 w-2 hover:bg-slate-500 dark:hover:bg-slate-500"
              }`}
              aria-label={`选择角色 ${character.name}`}
            />
          ))}
        </div>
      )}

      {/* 音频元素 */}
      <audio
        ref={audioRef}
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onEnded={() => setIsPlaying(false)}
        preload="auto"
      >
        <track
          kind="captions"
          src="data:text/vtt;base64,V0VCVlRUDQo="
          srcLang="zh"
          label="Chinese captions"
        />
      </audio>
    </div>
  );
}
