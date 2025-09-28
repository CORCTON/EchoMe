"use client";

import { useState, useMemo } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import { useMediaQuery } from "@/hooks/use-media-query";
import type { Character } from "@/types/character";

interface CharacterSearchProps {
  characters: Character[];
  onCharacterSelect?: (character: Character) => void;
}

export function CharacterSearch({
  characters,
  onCharacterSelect,
}: CharacterSearchProps) {
  const t = useTranslations("home");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const isDesktop = useMediaQuery("(min-width: 768px)");

  const filteredCharacters = useMemo(() => {
    if (!searchQuery.trim()) return characters;

    const query = searchQuery.toLowerCase();
    return characters.filter(
      (character) =>
        character.name.toLowerCase().includes(query) ||
        character.description?.toLowerCase().includes(query),
    );
  }, [characters, searchQuery]);

  const handleCharacterClick = (character: Character) => {
    if (onCharacterSelect) {
      onCharacterSelect(character);
      setOpen(false);
      setSearchQuery("");
      return;
    }

    setOpen(false);
    setSearchQuery("");
    router.push(`/${character.id}`);
  };

  const SearchContent = () => (
    <div className="flex flex-col space-y-4">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
        <Input
          placeholder={t("search_characters")}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10"
          autoFocus
        />
        {searchQuery && (
          <Button
            variant="ghost"
            size="sm"
            className="absolute right-2 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0"
            onClick={() => setSearchQuery("")}
          >
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>

      <div className="space-y-2 md:max-h-64 overflow-y-auto">
        {filteredCharacters.length === 0 ? (
          <div className="text-center py-8 text-gray-500">
            {searchQuery ? t("no_characters_found") : t("no_characters")}
          </div>
        ) : (
          filteredCharacters.map((character) => (
            <button
              key={character.id}
              type="button"
              className="flex items-center space-x-3 py-3 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer transition-colors w-full text-left"
              onClick={() => handleCharacterClick(character)}
            >
              <div className="flex-1 min-w-0">
                <p className="font-medium text-gray-900 dark:text-gray-100 truncate">
                  {character.name}
                </p>
                {character.description && (
                  <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
                    {character.description}
                  </p>
                )}
              </div>
            </button>
          ))
        )}
      </div>
    </div>
  );

  const trigger = (
    <Button
      variant="ghost"
      size="icon"
      className="rounded-full"
      aria-label={t("search_characters")}
    >
      <Search className="h-5 w-5" />
    </Button>
  );

  if (isDesktop) {
    return (
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>{trigger}</DialogTrigger>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("search_characters")}</DialogTitle>
          </DialogHeader>
          <SearchContent />
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Drawer
      open={open}
      onOpenChange={setOpen}
      setBackgroundColorOnScale={false}
      shouldScaleBackground
    >
      <DrawerTrigger asChild>{trigger}</DrawerTrigger>
      <DrawerContent>
        <DrawerHeader className="text-left">
          <DrawerTitle>{t("search_characters")}</DrawerTitle>
        </DrawerHeader>
        <div className="px-4 pb-4">
          <SearchContent />
        </div>
      </DrawerContent>
    </Drawer>
  );
}
