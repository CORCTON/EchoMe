"use client";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  useVoiceConversation,
  type MessageContent as MessageContentType,
  type TextContent,
} from "@/store/voice-conversation";
import { useCharacterStore } from "@/store/character";
import { useFileStore, type FileObject } from "@/store/file";

import {
  Message,
  MessageActions,
  MessageAvatar,
  MessageContent,
} from "@/components/ui/message";
import { MessageActionsComponent } from "@/components/ui/message-actions";
import { AudioAnimation } from "@/components/AudioAnimation";
import { VoiceActivity } from "@/types/vad";
import { Input } from "@/components/ui/input";
import { cn, getMimeTypeFromUrl } from "@/lib/utils";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";

export default function Page() {
  const { modelSettings, currentCharacter } = useCharacterStore();
  const characterId = currentCharacter?.id;
  const { setFiles } = useFileStore();

  const [isConversationStarted, setIsConversationStarted] = useState(false);
  const [editingMessageIndex, setEditingMessageIndex] = useState<number | null>(
    null,
  );
  const [input, setInput] = useState("");
  const [userHasScrolled, setUserHasScrolled] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const {
    history,
    start,
    pushUserMessage,
    isPlaying,
    resumeAudio,
    connect: connectLLM,
    interrupt,
    connection,
    editMessage,
    clear: clearHistory,
    setFiles: setConversationFiles,
  } = useVoiceConversation();

  useEffect(() => {
    if (!characterId) return;

    clearHistory();
    connectLLM(characterId);
    setIsConversationStarted(true);

    const filesToSet: FileObject[] = (modelSettings.fileUrls || []).map(
      (url) => ({
        url,
        name: url.split("/").pop() || "file",
        type: getMimeTypeFromUrl(url),
      }),
    );

    setFiles(filesToSet);
    setConversationFiles(filesToSet);
  }, [characterId, clearHistory, connectLLM, modelSettings.fileUrls, setFiles, setConversationFiles]);

  useEffect(() => {
    if (scrollRef.current && !userHasScrolled) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  });

  useEffect(() => {
    const scrollElement = scrollRef.current;
    const handleUserScroll = () => {
      setUserHasScrolled(true);
    };

    if (scrollElement && !userHasScrolled) {
      scrollElement.addEventListener("wheel", handleUserScroll);
      scrollElement.addEventListener("touchstart", handleUserScroll);
    }

    return () => {
      if (scrollElement) {
        scrollElement.removeEventListener("wheel", handleUserScroll);
        scrollElement.removeEventListener("touchstart", handleUserScroll);
      }
    };
  }, [userHasScrolled]);

  const handleSendMessage = () => {
    if (input.trim()) {
      pushUserMessage(input);
      const latestHistory = useVoiceConversation.getState().history;
      const messages = [
        {
          role: "system" as const,
          content: modelSettings.rolePrompt || "You are a helpful assistant.",
        },
        ...latestHistory,
      ];
      resumeAudio();
      start({ messages });
      setInput("");
    }
  };

  const getTextFromContent = (content: MessageContentType): string => {
    if (typeof content === "string") {
      return content;
    }
    const textPart = content.find((part) => part.type === "text") as TextContent | undefined;
    return textPart?.text || "";
  };

  const isUiReady = useMemo(() => {
    return isConversationStarted;
  }, [isConversationStarted]);

  const connectionStatusColor = useMemo(() => {
    switch (connection) {
      case "connected":
        return "bg-green-500";
      case "connecting":
        return "bg-yellow-500";
      default:
        return "bg-red-500";
    }
  }, [connection]);

  return (
    <div className=" bg-gradient-to-br from-slate-50 to-slate-200 dark:from-slate-900 dark:to-slate-800  flex items-center justify-center w-full border-none">
      <div className="w-full h-[100dvh] flex flex-col justify-center items-center relative">
        {currentCharacter && (
          <div className="absolute top-0 left-0 right-0 z-30 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm border-b border-slate-200 dark:border-slate-700">
            <div className="flex items-center justify-center py-3 px-4">
              <div className="flex items-center space-x-3">
                <div className="flex items-center space-x-2">
                  <h1 className="text-lg font-semibold text-slate-800 dark:text-slate-200">
                    {currentCharacter.name}
                  </h1>
                  <div
                    className={cn(
                      "w-3 h-3 rounded-full transition-colors duration-300",
                      connectionStatusColor,
                    )}
                    title={`Connection: ${connection}`}
                  />
                </div>
              </div>
            </div>
          </div>
        )}

        <div
          ref={scrollRef}
          className={cn(
            "w-full flex-1 overflow-y-auto pb-40 no-scrollbar [mask-image:linear-gradient(to_bottom,black_calc(100%-10rem),transparent)] transition-all duration-300 ease-in-out pointer-events-auto relative z-0",
            currentCharacter ? "pt-20" : "pt-4",
            { "opacity-60 blur-sm": !isUiReady },
          )}
        >
          <div className="px-4 space-y-4 mx-auto w-full max-w-4xl sm:px-6 md:px-8">
            {history
              .filter((msg) => msg.role !== "system")
              .map((msg, index) => {
                const isLastAssistantMessage = msg.role === "assistant" &&
                  index ===
                    history.filter((m) => m.role !== "system").length - 1;
                const originalMessageIndex = history.indexOf(msg);

                const handleSaveEdit = (newContent: string) => {
                  editMessage(originalMessageIndex, newContent, true);
                  setEditingMessageIndex(null);

                  setTimeout(() => {
                    const { history: updatedHistory } = useVoiceConversation
                      .getState();
                    const messages = [
                      {
                        role: "system" as const,
                        content: modelSettings.rolePrompt || "You are a helpful assistant.",
                      },
                      ...updatedHistory,
                    ];
                    start({ messages });
                  }, 100);
                };

                return (
                  <Message
                    key={`msg-${originalMessageIndex}-${msg.role}`}
                    className={cn("items-start gap-4 group", {
                      "justify-end": msg.role === "user",
                      "justify-start": msg.role === "assistant",
                    })}
                  >
                    {msg.role === "assistant" && currentCharacter && (
                      <MessageAvatar
                        src={currentCharacter.avatar}
                        alt={currentCharacter.name}
                        fallback={currentCharacter.name.charAt(0)}
                      />
                    )}
                    <div
                      className={cn(
                        "flex flex-col gap-2",
                        msg.role === "user" ? "max-w-[70%]" : "flex-1 min-w-0",
                      )}
                    >
                      {editingMessageIndex === originalMessageIndex
                        ? (
                          <Textarea
                            className="min-w-[40vh]"
                            defaultValue={getTextFromContent(msg.content)}
                            ref={(input) => input?.focus()}
                            onBlur={(e) => handleSaveEdit(e.target.value)}
                            onKeyDown={(e) => {
                              if (e.key === "Enter" && !e.shiftKey) {
                                e.preventDefault();
                                handleSaveEdit(e.currentTarget.value);
                              }
                            }}
                          />
                        )
                        : (
                          <MessageContent
                            id={`msg-${originalMessageIndex}-${msg.role}`}
                            markdown
                            className={cn(
                              {
                                "bg-white": msg.role === "user",
                              },
                              {
                                "bg-transparent p-0": msg.role === "assistant",
                              },
                            )}
                          >
                            {getTextFromContent(msg.content)}
                          </MessageContent>
                        )}
                      <MessageActions
                        className={cn("self-end", {
                          "self-start": msg.role === "assistant",
                        })}
                      >
                        <MessageActionsComponent
                          messageIndex={originalMessageIndex}
                          messageRole={msg.role as "user" | "assistant"}
                          isLastAssistantMessage={isLastAssistantMessage}
                          onEdit={() =>
                            setEditingMessageIndex(originalMessageIndex)}
                        />
                      </MessageActions>
                    </div>
                  </Message>
                );
              })}
          </div>
        </div>

        <div className="absolute bottom-10 left-1/2 -translate-x-1/2">
          <AudioAnimation
            activity={isPlaying ? VoiceActivity.Speaking : VoiceActivity.Idle}
          />
        </div>
        <div className="absolute bottom-8 left-4 right-4 flex items-center gap-4">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                handleSendMessage();
              }
            }}
            placeholder="Type your message..."
            className="flex-1"
          />
          <Button onClick={handleSendMessage}>Send</Button>
          <Button onClick={interrupt} variant="destructive">Interrupt</Button>
        </div>
      </div>
    </div>
  );
}
