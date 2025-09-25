export interface VoiceCharacter {
  id: string;
  name: string;
  image: string;
  description: {
    [key: string]: string;
  };
}

export const voiceCharacters: VoiceCharacter[] = [
  {
    id: "sol",
    name: "Sol",
    image: "https://api.dicebear.com/8.x/pixel-art/png?seed=sol&size=400",
    description: {
      zh: "阳光随性",
      en: "Sunny and casual",
    },
  },
  {
    id: "cove",
    name: "Cove",
    image: "https://api.dicebear.com/8.x/pixel-art/png?seed=cove&size=400",
    description: {
      zh: "沉稳内敛",
      en: "Calm and introverted",
    },
  },
  {
    id: "juniper",
    name: "Juniper",
    image: "https://api.dicebear.com/8.x/pixel-art/png?seed=juniper&size=400",
    description: {
      zh: "开放轻松",
      en: "Open and relaxed",
    },
  },
  {
    id: "sage",
    name: "Sage",
    image: "https://api.dicebear.com/8.x/pixel-art/png?seed=sage&size=400",
    description: {
      zh: "智慧温和",
      en: "Wise and gentle",
    },
  },
  {
    id: "nova",
    name: "Nova",
    image: "https://api.dicebear.com/8.x/pixel-art/png?seed=nova&size=400",
    description: {
      zh: "活力四射",
      en: "Energetic",
    },
  },
];

export function getCharacterById(id: string): VoiceCharacter | undefined {
  return voiceCharacters.find((character) => character.id === id);
}
