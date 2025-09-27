"use client";
import { create } from "zustand";
import { voiceCharacters, type VoiceCharacter } from "@/lib/characters";

export interface ModelSettings {
  rolePrompt?: string;
  fileUrls?: string[];
}

export interface CharacterState {
  characters: VoiceCharacter[];
  currentCharacter: VoiceCharacter | null;
  modelSettings: ModelSettings;
  setCurrentCharacter: (id: string) => void;
  updateModelSettings: (settings: ModelSettings) => void;
}

export const useCharacterStore = create<CharacterState>((set) => ({
  characters: voiceCharacters,
  currentCharacter: null,
  modelSettings: {},
  setCurrentCharacter: (id: string) =>
    set((state) => {
      if (state.currentCharacter?.id === id) {
        return state;
      }
      const character = voiceCharacters.find(
        (c: VoiceCharacter) => c.id === id,
      );
      return { currentCharacter: character || null };
    }),
  updateModelSettings: (settings: ModelSettings) => {
    set((state) => ({
      modelSettings: { ...state.modelSettings, ...settings },
    }));
  },
}));
