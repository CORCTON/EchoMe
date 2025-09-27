"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { UploadCloud, X } from "lucide-react";
import Image from "next/image";
import { useMutation } from '@tanstack/react-query';
import uploadFileService from '@/services/upload';

// PDF.js（通过 CDN 动态加载）声明
// biome-ignore lint/suspicious/noExplicitAny: PDF.js 来自 CDN，无类型定义
declare const pdfjsLib: any;

// 文件状态类型
interface FileState {
  id: string;
  previewUrl: string;
  fileName: string;
  progress: number;
  error?: string;
  isInitial: boolean;
}

interface ImageUploaderProps {
  onUploadComplete: (fileUrls: string[]) => void;
  initialFileUrls?: string[] | null;
}

export function ImageUploader({
  onUploadComplete,
  initialFileUrls,
}: ImageUploaderProps) {
  const t = useTranslations("home");
  const [files, setFiles] = useState<FileState[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [pdfjsLoaded, setPdfjsLoaded] = useState(false);

  // 工具：基于 File 创建 FileState
  const makeFileState = useCallback((file: File, id: string): FileState => ({
    id,
    previewUrl: URL.createObjectURL(file),
    fileName: file.name,
    progress: 0,
    isInitial: false,
  }), []);

  // 动态加载 PDF.js 库
  useEffect(() => {
    const script = document.createElement("script");
    script.src =
      "https://cdnjs.cloudflare.com/ajax/libs/pdf.js/4.4.168/pdf.min.mjs";
    script.type = "module";
    script.onload = () => {
      pdfjsLib.GlobalWorkerOptions.workerSrc =
        "https://cdnjs.cloudflare.com/ajax/libs/pdf.js/4.4.168/pdf.worker.min.mjs";
      setPdfjsLoaded(true);
    };
    document.body.appendChild(script);

    return () => {
      document.body.removeChild(script);
    };
  }, []);

  // 当初始文件 URL 变化时，更新文件列表
  useEffect(() => {
    const initialFiles =
      initialFileUrls?.map((url) => ({
        id: self.crypto.randomUUID(),
        previewUrl: url,
        fileName: url.split("/").pop() || "initial-file",
        progress: 100,
        isInitial: true,
      })) || [];
    setFiles(initialFiles);
  }, [initialFileUrls]);

  // 使用 react-query 的 mutation 统一处理上传
  const uploadMutation = useMutation<string, Error, File>({
    mutationFn: (file: File) => uploadFileService(file),
  });

  // 将 mutation 包装为稳定的上传函数，返回 url 或 null
  const doUpload = useCallback(
    async (file: File, fileStateId: string) => {
      try {
        const url = await uploadMutation.mutateAsync(file);
        setFiles((prev) => prev.map((f) => (f.id === fileStateId ? { ...f, previewUrl: url, progress: 100 } : f)));
        return url;
      } catch (err) {
        console.error('Upload error:', err);
        setFiles((prev) => prev.map((f) => (f.id === fileStateId ? { ...f, error: t('upload_failed') } : f)));
        return null;
      }
    },
    [uploadMutation, t],
  );

  // 处理用户选择的文件（图片或PDF）
  const handleFiles = useCallback(
    async (selectedFiles: FileList) => {
      const newFileStates: FileState[] = [];
      const uploadPromises: Promise<string | null>[] = [];
      const currentFileCount = files.length;
      const maxFiles = 3;
      let processedFileCount = 0;

      for (const file of Array.from(selectedFiles)) {
        if (currentFileCount + processedFileCount >= maxFiles) {
          console.warn(`Cannot upload more than ${maxFiles} files.`);
          break;
        }

        if (file.type.startsWith("image/")) {
          const id = self.crypto.randomUUID();
          newFileStates.push(makeFileState(file, id));
          uploadPromises.push(doUpload(file, id));
          processedFileCount++;
          continue;
        }

        if (file.type === "application/pdf") {
          if (!pdfjsLoaded) {
            console.warn("PDF.js 未就绪。");
            continue;
          }
          try {
            const pdf = await pdfjsLib.getDocument(URL.createObjectURL(file)).promise;
            for (let i = 1; i <= pdf.numPages; i++) {
              if (currentFileCount + processedFileCount >= maxFiles) {
                console.warn(`Cannot upload more than ${maxFiles} files.`);
                break;
              }
              const page = await pdf.getPage(i);
              const viewport = page.getViewport({ scale: 1.5 });
              const canvas = document.createElement("canvas");
              const ctx = canvas.getContext("2d");
              canvas.height = viewport.height;
              canvas.width = viewport.width;

              if (!ctx) continue;
              await page.render({ canvasContext: ctx, viewport }).promise;
              const blob: Blob | null = await new Promise((resolve) => canvas.toBlob(resolve, "image/png"));
              if (!blob) continue;

              const id = self.crypto.randomUUID();
              const pageFile = new File([blob], `${file.name}-page-${i}.png`, { type: "image/png" });
              newFileStates.push(makeFileState(pageFile, id));
              uploadPromises.push(doUpload(pageFile, id));
              processedFileCount++;
            }
          } catch (err) {
            console.error("PDF processing error:", err);
          }
        }
      }

      setFiles((prev) => [...prev, ...newFileStates]);

      Promise.all(uploadPromises).then((urls) => {
        const successful = urls.filter((u): u is string => !!u);
        // 避免在渲染期间执行父组件 setState：将回调放到微任务队列中异步执行
        const initial = initialFileUrls ?? [];
        const finalUrls = [...initial, ...successful];
        queueMicrotask(() => onUploadComplete(finalUrls));
      });
    },
    [pdfjsLoaded, onUploadComplete, doUpload, files.length, makeFileState, initialFileUrls],
  );

  // 移除一个文件
  const removeFile = (id: string) => {
    const fileToRemove = files.find((f) => f.id === id);
    if (!fileToRemove) return;

    // 从 state 中移除
    const updatedFiles = files.filter((f) => f.id !== id);
    setFiles(updatedFiles);

    // 如果被移除的文件不是初始文件，则从预览 URL 中撤销，释放内存
    if (!fileToRemove.isInitial) {
      URL.revokeObjectURL(fileToRemove.previewUrl);
    }

    // 更新父组件的状态
    const finalUrls = updatedFiles.map((f) => f.previewUrl);
    onUploadComplete(finalUrls);
  };

  return (
    <div className="space-y-2">
      <div className="space-y-0.5 px-4">
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
          multiple // 允许多文件选择
          className="hidden"
          onChange={(e) => e.target.files && handleFiles(e.target.files)}
        />

        {/* 文件预览列表 */}
        <div className="flex flex-col gap-2 px-4">
          {files.map((file) => (
            <div
              key={file.id}
              className="relative w-full h-28 rounded-md border border-dashed flex items-center justify-center"
            >
              <Image
                src={file.previewUrl}
                alt={file.fileName}
                fill
                sizes="100vw"
                className="object-contain rounded-md p-1"
              />
              <Button
                variant="ghost"
                size="icon"
                className="absolute top-0.5 right-0.5 h-5 w-5 bg-background/50 hover:bg-background/80"
                onClick={() => removeFile(file.id)}
              >
                <X className="h-3 w-3" />
              </Button>
              {file.progress > 0 && file.progress < 100 && (
                <div className="absolute bottom-0 left-0 w-full h-0.5 bg-muted rounded-b-md">
                  <div
                    className="h-0.5 bg-primary rounded-b-md"
                    style={{ width: `${file.progress}%` }}
                  />
                </div>
              )}
              {file.error && (
                <div className="absolute inset-0 bg-destructive/80 flex items-center justify-center text-destructive-foreground text-xs p-1 rounded-md">
                  {file.error}
                </div>
              )}
            </div>
          ))}
        </div>

        {/* 上传按钮，当文件数量小于3时显示 */}
        {files.length < 3 && (
          <div className="px-4 mt-2">
            <button
              type="button"
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
                if (e.dataTransfer.files) {
                  handleFiles(e.dataTransfer.files);
                }
              }}
              className={`w-full text-left rounded-md border-2 border-dashed p-4 text-sm text-muted-foreground cursor-pointer flex flex-col items-center justify-center h-28 transition-colors ${
                dragActive
                  ? "border-primary bg-primary/10"
                  : "border-input hover:border-primary/50"
              }`}
              aria-label={t("click_or_drag_to_upload")}
            >
              <UploadCloud className="h-8 w-8 text-muted-foreground mb-2" />
              <span className="text-center">
                {t("click_or_drag_to_upload")}
              </span>
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
