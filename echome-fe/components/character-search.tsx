"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { Search } from "lucide-react";
import { Button } from "@/components/ui/button";
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
  const isDesktop = useMediaQuery("(min-width: 768px)");

  const handleCharacterClick = (character: Character) => {
    if (onCharacterSelect) {
      onCharacterSelect(character);
      setOpen(false);
      return;
    }

    setOpen(false);
    router.push(`/${character.id}`);
  };

  const CharacterListContent = () => (
    <div className="flex flex-col">
      <div className="space-y-2 max-h-96 overflow-y-auto">
        {characters.length === 0 ? (
          <div className="text-center py-8 text-gray-500">
            {t("no_characters")}
          </div>
        ) : (
          characters.map((character) => (
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
      aria-label={t("select_character")}
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
            <DialogTitle>{t("select_character")}</DialogTitle>
          </DialogHeader>
          <CharacterListContent />
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
      <DrawerContent className="max-h-[80vh]">
        <DrawerHeader className="text-left">
          <DrawerTitle>{t("select_character")}</DrawerTitle>
        </DrawerHeader>
        <div className="px-4 pb-4 overflow-hidden">
          <CharacterListContent />
        </div>
      </DrawerContent>
    </Drawer>
  );
}
