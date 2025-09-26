"use client";

import { useCallback, useEffect, useRef, useState } from "react";
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
import type { VoiceCharacter } from "@/lib/characters";
import { Edit3, X } from "lucide-react";

interface ModelSettingsDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  character: VoiceCharacter;
  onReady: (settings: ModelSettings) => void;
}

export interface ModelSettings {
  fileUrl: string | null;
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
  const { modelSettings, updateModelSettings } = useCharacterStore();
  const [settings, setSettings] = useState<ModelSettings>({
    fileUrl: modelSettings.fileUrl ?? null,
    internetAccess: false, // This feature is not in the store yet
    rolePrompt: modelSettings.rolePrompt || character.prompt,
  });
  const [isUploading, setIsUploading] = useState(false);
  const [fileName, setFileName] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);

  const onFileChange = useCallback(async (file?: File | null) => {
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    if (!file) {
      setSettings((prev) => ({ ...prev, fileUrl: null }));
      setFileName(null);
      setFileError(null);
      return;
    }

    const ok = file.type.startsWith("image/") ||
      file.type === "application/pdf";
    if (!ok) {
      setFileError(t("file_type_error"));
      setFileName(null);
      return;
    }

    setFileError(null);
    setIsUploading(true);
    setFileName(file.name);

    const formData = new FormData();
    formData.append("file", file);

    try {
      const response = await fetch("/api/upload", {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        throw new Error("Upload failed");
      }

      const { url } = await response.json();
      setSettings((prev) => ({ ...prev, fileUrl: url }));
    } catch (error) {
      console.error("Upload error:", error);
      setFileError(t("upload_failed"));
      setFileName(null);
      setSettings((prev) => ({ ...prev, fileUrl: null }));
    } finally {
      setIsUploading(false);
    }
  }, [t]);
  const [isEditingPrompt, setIsEditingPrompt] = useState(false);
  const [isDesktop, setIsDesktop] = useState(false);

  useEffect(() => {
    const checkIsDesktop = () => {
      setIsDesktop(window.innerWidth >= 640); // sm breakpoint
    };

    checkIsDesktop();
    window.addEventListener("resize", checkIsDesktop);

    return () => window.removeEventListener("resize", checkIsDesktop);
  }, []);

  useEffect(() => {
    setSettings({
      fileUrl: modelSettings.fileUrl ?? null,
      internetAccess: false,
      rolePrompt: modelSettings.rolePrompt || character.prompt,
    });
    setFileName(null);
    setFileError(null);
    setIsEditingPrompt(false);
  }, [character, modelSettings]);

  const handleReady = () => {
    updateModelSettings(settings);
    onReady(settings);
    onOpenChange(false);
  };

  return (
    <Drawer
      open={open}
      onOpenChange={onOpenChange}
      direction={isDesktop ? "right" : "bottom"}
      shouldScaleBackground={!isDesktop}
      setBackgroundColorOnScale={false}
    >
      <DrawerContent
        className={isDesktop
          ? "h-full w-96 max-w-96 flex flex-col"
          : "h-[90vh] w-full flex flex-col"}
      >
        <DrawerHeader className="flex flex-row items-start justify-between pb-4">
          <div className="flex flex-col items-start">
            <DrawerTitle className="text-lg font-semibold">
              {character.name}
            </DrawerTitle>
          </div>
          <div className="ml-3 flex-shrink-0">
            <DrawerClose asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <X className="h-4 w-4" />
              </Button>
            </DrawerClose>
          </div>
        </DrawerHeader>

        {/* 内容区：flex-1 可滚动，确保 footer 固定 */}
        <div className="flex-1 overflow-y-auto p-4 space-y-6">
          {/* 文件上传设置：点击或拖拽上传 */}
          <div className="space-y-2">
            <div className="">
              <div className="space-y-0.5">
                <div className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                  {t("file_upload")}
                </div>
                <p className="text-xs text-muted-foreground">
                  {t("file_upload_description")}
                </p>
              </div>

              <div className="mt-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*,application/pdf"
                  className="hidden"
                  onChange={(e) => {
                    const f = (e.target as HTMLInputElement).files?.[0] ?? null;
                    onFileChange(f);
                  }}
                />

                {/** biome-ignore lint/a11y/useSemanticElements: 需要嵌套Button */}
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => fileInputRef.current?.click()}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      fileInputRef.current?.click();
                    }
                  }}
                  onDragOver={(e) => {
                    e.preventDefault();
                    setDragActive(true);
                  }}
                  onDragLeave={(e) => {
                    e.preventDefault();
                    setDragActive(false);
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    setDragActive(false);
                    const f = e.dataTransfer?.files?.[0] ?? null;
                    if (f) onFileChange(f);
                  }}
                  className={"w-full text-left rounded-md border border-dashed p-4 text-sm text-muted-foreground cursor-pointer " +
                    (dragActive
                      ? "bg-accent/5 border-accent"
                      : "bg-transparent")}
                  aria-label={t("click_or_drag_to_upload")}
                >
                  {isUploading
                    ? (
                      <div className="flex items-center justify-center gap-2">
                        <div className="text-sm text-muted-foreground">
                          {t("uploading")}...
                        </div>
                      </div>
                    )
                    : fileName
                    ? (
                      <div className="flex items-center justify-between">
                        <div className="truncate text-left">
                          {fileName}
                        </div>
                        <div className="ml-3 flex-shrink-0">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation();
                              onFileChange(null);
                            }}
                          >
                            {t("remove")}
                          </Button>
                        </div>
                      </div>
                    )
                    : (
                      <div className="flex items-center justify-center gap-2">
                        <div className="text-sm text-muted-foreground">
                          {t("click_or_drag_to_upload")}
                        </div>
                      </div>
                    )}
                </div>
              </div>

              {fileError && (
                <div className="mt-2 text-xs text-destructive">{fileError}</div>
              )}
            </div>
          </div>

          {/* 联网功能设置 */}
          <div className="space-y-2">
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
                  setSettings((prev) => ({ ...prev, internetAccess: checked }))}
              />
            </div>
          </div>

          {/* 角色提示词设置 */}
          <div className="space-y-3">
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

            {isEditingPrompt
              ? (
                <Textarea
                  value={settings.rolePrompt}
                  onChange={(e) =>
                    setSettings((prev) => ({
                      ...prev,
                      rolePrompt: e.target.value,
                    }))}
                  placeholder={character.prompt}
                  className="min-h-[120px] resize-none"
                />
              )
              : (
                <div className="p-3 bg-muted rounded-md text-sm text-muted-foreground">
                  {settings.rolePrompt}
                </div>
              )}
          </div>
        </div>

        {/* Footer 固定在底部并使用安全区内边距，移动端时保持可见 */}
        <DrawerFooter className="pt-4 bg-background">
          <div className="w-full bg-transparent">
            <Button
              onClick={handleReady}
              className="w-full cursor-pointer  rounded-2xl"
              size="lg"
            >
              {t("ready_button")}
            </Button>
          </div>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
