"use client";
import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Character } from "@/types/character";

export interface ModelSettings {
  rolePrompt?: string;
  fileUrls?: string[];
  internetAccess?: boolean;
}

export interface CharacterState {
  characters: Character[];
  currentCharacter: Character | null;
  modelSettings: ModelSettings;
  setCharacters: (characters: Character[]) => void;
  setCurrentCharacter: (id: string) => void;
  updateModelSettings: (settings: ModelSettings) => void;
}

export const useCharacterStore = create(
  persist<CharacterState>(
    (set) => ({
      characters: [],
      currentCharacter: null,
      modelSettings: {},
      setCharacters: (characters: Character[]) => set({ characters }),
      setCurrentCharacter: (id: string) =>
        set((state) => {
          if (state.currentCharacter?.id === id) {
            return state;
          }
          const character = state.characters.find((c: Character) => c.id === id);
          return { currentCharacter: character || null };
        }),
      updateModelSettings: (settings: ModelSettings) => {
        set((state) => ({
          modelSettings: { ...state.modelSettings, ...settings },
        }));
      },
    }),
    {
      name: "character-storage",
    },
  ),
);
