"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import useEmblaCarousel from "embla-carousel-react";
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
  const [playingId, setPlayingId] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const [emblaRef, emblaApi] = useEmblaCarousel({
    loop: characters.length > 1,
    align: "center",
    skipSnaps: false,
    dragFree: false,
  });

  const scrollDelta = useRef(0);
  const debounceTimeout = useRef<NodeJS.Timeout | null>(null);
  const debounceDuration = 300; // ms

  const applyScroll = useCallback(() => {
    if (!emblaApi || scrollDelta.current === 0) return;
    const currentIndex = emblaApi.selectedScrollSnap();
    const targetIndex = currentIndex + scrollDelta.current;
    emblaApi.scrollTo(targetIndex);
    scrollDelta.current = 0;
  }, [emblaApi]);

  const debouncedScroll = useCallback(
    (direction: "next" | "prev") => {
      if (debounceTimeout.current) {
        clearTimeout(debounceTimeout.current);
      }
      scrollDelta.current += direction === "next" ? 1 : -1;
      debounceTimeout.current = setTimeout(applyScroll, debounceDuration);
    },
    [applyScroll],
  );

  const scrollPrev = useCallback(
    () => debouncedScroll("prev"),
    [debouncedScroll],
  );
  const scrollNext = useCallback(
    () => debouncedScroll("next"),
    [debouncedScroll],
  );

  const scrollTo = useCallback(
    (index: number) => {
      if (emblaApi) emblaApi.scrollTo(index);
    },
    [emblaApi],
  );

  useEffect(() => {
    return () => {
      if (debounceTimeout.current) {
        clearTimeout(debounceTimeout.current);
      }
    };
  }, []);

  const onSelect = useCallback(() => {
    if (!emblaApi) return;
    const selectedIndex = emblaApi.selectedScrollSnap();
    onCharacterSelect(characters[selectedIndex]);
  }, [emblaApi, characters, onCharacterSelect]);

  useEffect(() => {
    if (!emblaApi) return;

    onSelect();
    emblaApi.on("select", onSelect);

    return () => {
      emblaApi.off("select", onSelect);
    };
  }, [emblaApi, onSelect]);

  // 同步外部选择的角色到轮播
  useEffect(() => {
    if (!emblaApi) return;

    const currentIndex = characters.findIndex(
      (char) => char.id === selectedCharacter.id,
    );
    if (currentIndex !== -1 && currentIndex !== emblaApi.selectedScrollSnap()) {
      emblaApi.scrollTo(currentIndex);
    }
    if (audioRef.current) {
      const voiceSrc = (selectedCharacter.audio_example || "").trim();
      audioRef.current.src = voiceSrc !== "null" ? voiceSrc : "";
    }
  }, [emblaApi, characters, selectedCharacter]);

  return (
    <div className="relative w-full max-w-[90vw] mx-auto px-[2vw] sm:px-[3vw]">
      {/* 轮播容器 - 使用 vh 单位 */}
      <div className=" h-[50vh] sm:h-[45vh] md:h-[40vh]" ref={emblaRef}>
        <div
          className={`flex h-full ${
            characters.length < 3 ? "justify-center" : ""
          }`}
        >
          {characters.map((character, index) => {
            const isSelected = character.id === selectedCharacter.id;
            const voiceSrc = (character.audio_example || "").trim();
            const hasValidVoice = voiceSrc && voiceSrc !== "null";

            return (
              <div
                key={character.id}
                className={`flex-[0_0_100%] ${
                  characters.length > 1
                    ? "sm:flex-[0_0_50%] md:flex-[0_0_33.333%]"
                    : ""
                } min-w-0 px-[1vw] sm:px-[2vw] h-full`}
              >
                <div className="flex flex-col items-center justify-start h-full">
                  {/* 角色图像容器 - 使用 vw/vh 单位 */}
                  <div className="relative flex items-center justify-center h-[35vh] sm:h-[32vh] md:h-[30vh] mb-[2vh]">
                    <button
                      type="button"
                      className={`relative rounded-full overflow-hidden border-4 shadow-2xl cursor-pointer focus:outline-none focus:ring-2 focus:ring-slate-400 transition-all duration-500 ${
                        isSelected
                          ? "w-[35vw] h-[35vw] sm:w-[25vw] sm:h-[25vw] md:w-[15vw] md:h-[15vw] border-white dark:border-slate-700 opacity-100 scale-100"
                          : "w-[20vw] h-[20vw] sm:w-[15vw] sm:h-[15vw] md:w-[10vw] md:h-[10vw] border-slate-300 dark:border-slate-600 opacity-60 scale-90 hover:opacity-80 hover:scale-95"
                      }`}
                      style={{
                        maxWidth: isSelected ? "200px" : "120px",
                        maxHeight: isSelected ? "200px" : "120px",
                      }}
                      onClick={() => scrollTo(index)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          scrollTo(index);
                        }
                      }}
                      aria-label={`Select ${character.name}`}
                    >
                      {character.avatar ? (
                        <Image
                          src={character.avatar}
                          alt={character.name}
                          className="w-full h-full object-cover transition-all duration-500"
                          width={256}
                          height={256}
                          priority={isSelected}
                        />
                      ) : (
                        <div className="w-full h-full bg-slate-200 dark:bg-slate-700" />
                      )}
                      <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent" />
                    </button>
                    {hasValidVoice && isSelected && (
                      <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                        <button
                          type="button"
                          onClick={() => {
                            if (playingId === character.id) {
                              audioRef.current?.pause();
                            } else {
                              if (audioRef.current?.src) {
                                audioRef.current.play();
                                setPlayingId(character.id);
                              }
                            }
                          }}
                          className="p-2 bg-black/50 rounded-full text-white pointer-events-auto"
                          aria-label={`Play voice for ${character.name}`}
                        >
                          {isPlaying && playingId === character.id ? (
                            <Pause size={24} />
                          ) : (
                            <Play size={24} />
                          )}
                        </button>
                      </div>
                    )}
                  </div>

                  {/* 角色信息 - 使用 vh 单位 */}
                  <div className="text-center h-[10vh] sm:h-[8vh] md:h-[6vh] flex flex-col justify-center">
                    <h2
                      className={`text-lg sm:text-xl font-semibold mb-1 transition-opacity duration-300 ${
                        isSelected
                          ? "text-slate-800 dark:text-slate-200 opacity-100"
                          : "text-transparent opacity-0"
                      }`}
                    >
                      {character.name}
                    </h2>
                    <p
                      className={`text-xs sm:text-sm transition-opacity duration-300 ${
                        isSelected
                          ? "text-slate-600 dark:text-slate-400 opacity-100"
                          : "text-transparent opacity-0"
                      }`}
                    >
                      {character.description}
                    </p>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
      <audio
        ref={audioRef}
        onPlay={() => setIsPlaying(true)}
        onPause={() => {
          setIsPlaying(false);
          setPlayingId(null);
        }}
        onEnded={() => {
          setIsPlaying(false);
          setPlayingId(null);
        }}
        preload="auto"
      >
        <track
          kind="captions"
          src="data:text/vtt;base64,V0VCVlRUDQo="
          srcLang="en"
          label="English captions"
        />
      </audio>

      {/* 导航按钮 - 使用 vw/vh 单位定位 */}
      {characters.length > 1 && (
        <>
          <Button
            variant="outline"
            size="icon"
            onClick={scrollPrev}
            className="cursor-pointer absolute left-[2vw] sm:left-[3vw] md:left-4 top-[15vh] sm:top-[12vh] md:top-1/2 md:-translate-y-1/2 rounded-full bg-white/90 dark:bg-slate-800/90 backdrop-blur-sm hover:bg-white dark:hover:bg-slate-800 transition-all duration-200 z-10 shadow-lg"
            style={{
              width: "min(8vw, 40px)",
              height: "min(8vw, 40px)",
            }}
          >
            <ChevronLeft size={16} className="sm:w-5 sm:h-5" />
          </Button>

          <Button
            variant="outline"
            size="icon"
            onClick={scrollNext}
            className="cursor-pointer  absolute right-[2vw] sm:right-[3vw] md:right-4 top-[15vh] sm:top-[12vh] md:top-1/2 md:-translate-y-1/2 rounded-full bg-white/90 dark:bg-slate-800/90 backdrop-blur-sm hover:bg-white dark:hover:bg-slate-800 transition-all duration-200 z-10 shadow-lg"
            style={{
              width: "min(8vw, 40px)",
              height: "min(8vw, 40px)",
            }}
          >
            <ChevronRight size={16} className="sm:w-5 sm:h-5" />
          </Button>
        </>
      )}

      {/* 指示器 */}
      {characters.length > 1 && (
        <div className="flex justify-center space-x-2 mt-6">
          {characters.map((character, index) => (
            <Button
              key={character.id}
              variant="ghost"
              onClick={() => scrollTo(index)}
              className={`cursor-pointer h-2 rounded-full transition-all duration-300 p-0 hover:scale-110 ${
                character.id === selectedCharacter.id
                  ? "bg-slate-800 dark:bg-slate-200 w-8"
                  : "bg-slate-400 dark:bg-slate-600 w-2 hover:bg-slate-500 dark:hover:bg-slate-500"
              }`}
            />
          ))}
        </div>
      )}
    </div>
  );
}
