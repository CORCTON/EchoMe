"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { getCharacters } from "@/services/character";

import { useVadStore } from "@/store/vad";
import { VoiceActivity } from "@/types/vad";
import {
  useVoiceConversation,
  type MessageContent as MessageContentType,
  type TextContent,
} from "@/store/voice-conversation";
import { useCharacterStore } from "@/store/character";
import type { FileObject } from "@/store/file";

import {
  Message,
  MessageActions,
  MessageAvatar,
  MessageContent,
} from "@/components/ui/message";
import { MessageActionsComponent } from "@/components/ui/message-actions";
import { AudioAnimation } from "@/components/AudioAnimation";
import { cn, getMimeTypeFromUrl } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { Loader } from "@/components/ui/loader";
import { Textarea } from "@/components/ui/textarea";

export default function Page() {
  const params = useParams<{ id: string }>();
  const characterId = params?.id ?? "";

  const {
    currentCharacter,
    setCurrentCharacter,
    modelSettings,
    setCharacters: setStoreCharacters,
  } = useCharacterStore();

  const [isConversationStarted, setIsConversationStarted] = useState(false);
  const [editingMessageIndex, setEditingMessageIndex] = useState<number | null>(
    null,
  );
  const [userHasScrolled, setUserHasScrolled] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const {
    isVadReady,
    voiceActivity,
    transcript,
    initVad,
    resetTranscript,
    forceEndSpeech,
  } = useVadStore();
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
    isResponding,
    clear: clearHistory,
    setFiles,
  } = useVoiceConversation();

  useEffect(() => {
    const fetchCharacters = async () => {
      try {
        const response = await getCharacters();
        if (response.success) {
          setStoreCharacters(response.data);
        }
      } catch (error) {
        console.error("Failed to fetch characters", error);
      }
    };
    fetchCharacters();
  }, [setStoreCharacters]);

  useEffect(() => {
    const onSpeechEnd = (transcript: string) => {
      if (transcript && characterId && currentCharacter) {
        resumeAudio();
        pushUserMessage(transcript);
        const latestHistory = useVoiceConversation.getState().history;
        const messages = [
          {
            role: "system" as const,
            content:
              modelSettings.rolePrompt ||
              currentCharacter?.prompt ||
              "You are a helpful assistant.",
          },
          ...latestHistory,
        ];
        start({ messages });
        resetTranscript();
      }
    };

    initVad(onSpeechEnd);
    return () => {
      // 停止播放并断开连接
      const { stopPlaying, disconnect: disconnectLLM } =
        useVoiceConversation.getState();
      stopPlaying();
      disconnectLLM();

      // 断开VAD
      const { disconnect: disconnectVad } = useVadStore.getState();
      disconnectVad();
    };
  }, [
    initVad,
    characterId,
    pushUserMessage,
    start,
    resetTranscript,
    resumeAudio,
    currentCharacter,
    modelSettings.rolePrompt,
  ]);

  useEffect(() => {
    if (isVadReady && characterId) {
      setCurrentCharacter(characterId);
      clearHistory();

      // 设置文件
      const filesToSet: FileObject[] = (modelSettings.fileUrls || []).map(
        (url) => ({
          url,
          name: url.split("/").pop() || "file",
          type: getMimeTypeFromUrl(url),
        }),
      );
      setFiles(filesToSet);

      resumeAudio();
      connectLLM(characterId);
      setIsConversationStarted(true);
    }
  }, [
    isVadReady,
    characterId,
    resumeAudio,
    connectLLM,
    setCurrentCharacter,
    clearHistory,
    modelSettings.fileUrls,
    setFiles,
  ]);

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

  useEffect(() => {
    if (voiceActivity === VoiceActivity.Speaking && isPlaying) {
      interrupt();
    }
  }, [voiceActivity, interrupt, isPlaying]);

  const isUiReady = useMemo(() => {
    return isVadReady && isConversationStarted;
  }, [isVadReady, isConversationStarted]);

  const t = useTranslations("home");

  const handleInterrupt = () => {
    if (voiceActivity === VoiceActivity.Speaking) {
      forceEndSpeech();
    }
  };

  const animationActivity = useMemo(() => {
    if (isResponding) {
      return VoiceActivity.Loading;
    }
    if (!isVadReady || !isConversationStarted) {
      return VoiceActivity.Loading;
    }
    if (isPlaying) {
      return VoiceActivity.Idle;
    }
    if (voiceActivity === VoiceActivity.Loading) {
      return VoiceActivity.Idle;
    }
    return voiceActivity;
  }, [
    isVadReady,
    isConversationStarted,
    voiceActivity,
    isPlaying,
    isResponding,
  ]);

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
                const isLastAssistantMessage =
                  msg.role === "assistant" &&
                  index ===
                    history.filter((m) => m.role !== "system").length - 1;
                const originalMessageIndex = history.indexOf(msg);

                const handleSaveEdit = (newContent: string) => {
                  editMessage(originalMessageIndex, newContent, true);
                  setEditingMessageIndex(null);

                  setTimeout(() => {
                    const { history: updatedHistory } =
                      useVoiceConversation.getState();
                    if (characterId && currentCharacter) {
                      const messages = [
                        {
                          role: "system" as const,
                          content:
                            modelSettings.rolePrompt ||
                            currentCharacter?.prompt ||
                            "You are a helpful assistant.",
                        },
                        ...updatedHistory,
                      ];
                      start({ messages });
                    }
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
                      {editingMessageIndex === originalMessageIndex ? (
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
                      ) : (
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
                            setEditingMessageIndex(originalMessageIndex)
                          }
                        />
                      </MessageActions>
                    </div>
                  </Message>
                );
              })}
            {voiceActivity === VoiceActivity.Speaking && transcript && (
              <Message className="items-start justify-end gap-4">
                <MessageContent className="bg-white">
                  {transcript}
                </MessageContent>
              </Message>
            )}
          </div>
        </div>

        <div className="absolute inset-0 pointer-events-none z-10">
          {!isVadReady && (
            <div className="absolute inset-0 flex justify-center items-center">
              <Loader />
              <span className="sr-only">{t("vad_loading")}</span>
            </div>
          )}
          {isVadReady &&
            isConversationStarted &&
            animationActivity === VoiceActivity.Idle &&
            history.length === 0 && (
              <div className="absolute inset-0 flex justify-center items-center pointer-events-none">
                <p className="text-lg text-gray-600">{t("say_something")}</p>
              </div>
            )}

          {/** biome-ignore lint/a11y/useSemanticElements: 需要嵌套button*/}
          <div
            role="button"
            tabIndex={0}
            className={cn(
              "absolute bottom-10 left-1/2 -translate-x-1/2 w-40 h-40 transition-opacity duration-300 ease-in-out cursor-pointer pointer-events-auto",
              { "opacity-0": !isVadReady },
            )}
            onClick={handleInterrupt}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleInterrupt();
              }
            }}
            aria-label={
              voiceActivity === VoiceActivity.Speaking
                ? t("tap_to_interrupt")
                : undefined
            }
          >
            <AudioAnimation activity={animationActivity} />
            {voiceActivity === VoiceActivity.Speaking && (
              <p className="text-xs text-center text-gray-500 mt-2">
                {t("tap_to_interrupt")}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function getTextFromContent(content: MessageContentType): string {
  if (typeof content === "string") {
    return content;
  }
  const textPart = content.find((part) => part.type === "text") as
    | TextContent
    | undefined;
  return textPart?.text || "";
}
