"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { useCharacterStore } from "@/store/character";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";

import { Edit3, X } from "lucide-react";
import { ImageUploader } from "./image-uploader";
import { useFileStore } from "@/store/file";
import { useRouter } from "next/navigation";
import type { Character } from "@/types/character";
import { getMimeTypeFromUrl } from "@/lib/utils";

interface ModelSettingsDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  character: Character;
  onReady: (settings: Omit<ModelSettings, "fileUrls">) => void;
}

export interface ModelSettings {
  fileUrls: string[];
  internetAccess: boolean;
  rolePrompt: string;
}

export function ModelSettingsDrawer({
  open,
  onOpenChange,
  character,
  onReady,
}: ModelSettingsDrawerProps) {
  const t = useTranslations("home");
  const router = useRouter();
  const { modelSettings, updateModelSettings } = useCharacterStore();
  const { files, addFile, clearFiles } = useFileStore();
  const [settings, setSettings] = useState<Omit<ModelSettings, "fileUrls">>({
    internetAccess: false,
    rolePrompt: modelSettings.rolePrompt || character.prompt,
  });
  const [isEditingPrompt, setIsEditingPrompt] = useState(false);
  const [isDesktop, setIsDesktop] = useState(false);
  const [isUploading, setIsUploading] = useState(false);

  useEffect(() => {
    const checkIsDesktop = () => {
      setIsDesktop(window.innerWidth >= 640);
    };

    checkIsDesktop();
    window.addEventListener("resize", checkIsDesktop);

    return () => window.removeEventListener("resize", checkIsDesktop);
  }, []);

  useEffect(() => {
    const defaultSettings = {
      internetAccess: false,
      rolePrompt: character.prompt,
      fileUrls: [],
    };

    setSettings({
      internetAccess: defaultSettings.internetAccess,
      rolePrompt: defaultSettings.rolePrompt,
    });

    updateModelSettings(defaultSettings);
    clearFiles();
    setIsEditingPrompt(false);
  }, [character, clearFiles, updateModelSettings]);

  const handleReady = () => {
    const finalSettings = { ...settings, fileUrls: files.map((f) => f.url) };
    updateModelSettings(finalSettings);
    onReady(settings);
    onOpenChange(false);
  };

  const handleGoToTestPage = () => {
    const finalSettings = { ...settings, fileUrls: files.map((f) => f.url) };
    updateModelSettings(finalSettings);
    // All data is now in the store, no need for URL params
    router.push(`/test`);
    onOpenChange(false);
  };

  return (
    <Drawer
      dismissible={false}
      open={open}
      onOpenChange={onOpenChange}
      direction={isDesktop ? "right" : "bottom"}
      shouldScaleBackground={!isDesktop}
      setBackgroundColorOnScale={false}
    >
      <DrawerContent
        className={
          isDesktop
            ? "h-full w-96 max-w-96 flex flex-col"
            : "h-[90vh] w-full flex flex-col"
        }
      >
        <DrawerHeader className="flex flex-row items-start justify-between pb-4">
          <div className="flex flex-col items-start">
            <DrawerTitle className="text-lg font-semibold">
              {character.name}
            </DrawerTitle>
          </div>
          <div className="ml-3 flex-shrink-0">
            <DrawerClose asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={() => onOpenChange(false)}
              >
                <X className="h-4 w-4" />
              </Button>
            </DrawerClose>
          </div>
        </DrawerHeader>

        {/* 内容区：flex-1 可滚动，确保 footer 固定 */}
        <div className="flex-1 overflow-y-auto space-y-6">
          <ImageUploader
            initialFileUrls={files.map((f) => f.url)}
            onUploadComplete={(urls) => {
              clearFiles();
              urls.forEach((url) => {
                addFile({
                  url,
                  name: url.split("/").pop() || "uploaded-file",
                  type: getMimeTypeFromUrl(url),
                });
              });
            }}
            onUploadStatusChange={setIsUploading}
          />

          {/* 联网功能设置 */}
          <div className="space-y-2 px-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <div className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                  {t("internet_access")}
                </div>
                <p className="text-xs text-muted-foreground">
                  {t("internet_access_description")}
                </p>
              </div>
              <Switch
                checked={settings.internetAccess}
                onCheckedChange={(checked) =>
                  setSettings((prev) => ({ ...prev, internetAccess: checked }))
                }
              />
            </div>
          </div>

          {/* 角色提示词设置 */}
          <div className="space-y-3 px-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <div className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                  {t("role_prompt")}
                </div>
                <p className="text-xs text-muted-foreground">
                  {t("role_prompt_description")}
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setIsEditingPrompt(!isEditingPrompt)}
                className="h-8 px-2 cursor-pointer"
              >
                <Edit3 className="h-3 w-3 mr-1" />
                {t("edit_prompt")}
              </Button>
            </div>

            {isEditingPrompt ? (
              <Textarea
                value={settings.rolePrompt}
                onChange={(e) =>
                  setSettings((prev) => ({
                    ...prev,
                    rolePrompt: e.target.value,
                  }))
                }
                placeholder={character.prompt}
                className="min-h-[120px] resize-none"
              />
            ) : (
              <div className="p-3 bg-muted rounded-md text-sm text-muted-foreground">
                {settings.rolePrompt}
              </div>
            )}
          </div>
        </div>

        {/* Footer 固定在底部并使用安全区内边距，移动端时保持可见 */}
        <DrawerFooter className="pt-4 bg-background">
          <div className="w-full bg-transparent space-y-2">
            <Button
              onClick={handleReady}
              disabled={isUploading}
              className="w-full cursor-pointer rounded-2xl"
              size="lg"
            >
              {isUploading ? "文件上传中..." : t("ready_button")}
            </Button>
            <Button
              onClick={handleGoToTestPage}
              disabled={isUploading}
              variant="outline"
              className="w-full cursor-pointer rounded-2xl"
              size="lg"
            >
              {isUploading ? "文件上传中..." : "进入测试页面"}
            </Button>
          </div>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
