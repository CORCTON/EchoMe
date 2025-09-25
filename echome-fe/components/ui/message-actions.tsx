"use client";

import { useState } from "react";
import { Edit3, Trash2, RotateCcw } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { MessageAction } from "@/components/ui/message";
import { useVoiceConversation } from "@/store/voice-conversation";
import { useTranslations } from "next-intl";

interface MessageActionsProps {
  messageIndex: number;
  messageRole: "user" | "assistant";
  isLastAssistantMessage?: boolean;
  onEdit: () => void;
}

export function MessageActionsComponent({
  messageIndex,
  messageRole,
  isLastAssistantMessage = false,
  onEdit,
}: MessageActionsProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const { deleteMessage, retryLastAssistantMessage } = useVoiceConversation();
  const t = useTranslations("home");

  const handleDelete = () => {
    deleteMessage(messageIndex);
    setDeleteDialogOpen(false);
  };

  const handleRetry = () => {
    retryLastAssistantMessage();
  };

  return (
    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-auto">
      {messageRole === "user" && (
        <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
          <AlertDialogTrigger asChild>
            <MessageAction tooltip={t("delete_message")}>
              <Button variant="ghost" size="sm" className="h-6 w-6 p-0">
                <Trash2 className="h-3 w-3" />
              </Button>
            </MessageAction>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t("confirm_delete")}</AlertDialogTitle>
              <AlertDialogDescription>
                {t("delete_message_warning")}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
              <AlertDialogAction onClick={handleDelete}>
                {t("delete")}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}

      {messageRole === "user" && (
        <MessageAction tooltip={t("edit_message")}>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={onEdit}
          >
            <Edit3 className="h-3 w-3" />
          </Button>
        </MessageAction>
      )}

      {messageRole === "assistant" && isLastAssistantMessage && (
        <MessageAction tooltip={t("retry")}>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={handleRetry}
          >
            <RotateCcw className="h-3 w-3" />
          </Button>
        </MessageAction>
      )}
    </div>
  );
}
