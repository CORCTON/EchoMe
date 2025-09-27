import { getCharacters } from "@/services/character";
import HomePage from "./home-page";

export default async function Home() {
  const charactersResponse = await getCharacters();
  const characters = charactersResponse.data ?? [];

  return <HomePage initialCharacters={characters} />;
}
