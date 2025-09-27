import type { APIResponse } from "@/types/api";
import type { Character } from "@/types/character";

export const getCharacters = async (): Promise<APIResponse<Character[]>> => {
  let url: string;
  if (typeof window !== "undefined") {
    // 客户端环境
    url = "/v1/api/characters";
  } else {
    // 服务端环境
    const baseUrl = process.env.API_BASE_URL;
    url = `${baseUrl}/api/characters`;
  }
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error("Failed to fetch characters");
  }
  return response.json();
};
