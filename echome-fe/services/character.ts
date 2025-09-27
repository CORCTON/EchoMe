import type { APIResponse } from "@/types/api";
import type { Character } from "@/types/character";

export const getCharacters = async (): Promise<APIResponse<Character[]>> => {
  const baseUrl = process.env.API_BASE_URL;
  const response = await fetch(`${baseUrl}/api/characters`);
  if (!response.ok) {
    throw new Error("Failed to fetch characters");
  }
  return response.json();
};
