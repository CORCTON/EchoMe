"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";

import { useVadStore } from "@/store/vad";
import { VoiceActivity } from "@/types/vad";
import { useVoiceConversation } from "@/store/voice-conversation";
import { getCharacterById } from "@/lib/characters";

import {
  Message,
  MessageActions,
  MessageAvatar,
  MessageContent,
} from "@/components/ui/message";
import { MessageActionsComponent } from "@/components/ui/message-actions";
import { AudioAnimation } from "@/components/AudioAnimation";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { Loader } from "@/components/ui/loader";
import { Textarea } from "@/components/ui/textarea";

export default function Page() {
  const params = useParams<{ id: string }>();
  const characterId = params?.id ?? "";

  const [isConversationStarted, setIsConversationStarted] = useState(false);
  const [editingMessageIndex, setEditingMessageIndex] = useState<number | null>(
    null,
  );
  const scrollRef = useRef<HTMLDivElement>(null);

  const { isVadReady, voiceActivity, transcript, initVad, resetTranscript } =
    useVadStore();
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
  } = useVoiceConversation();

  const currentCharacter = useMemo(
    () => getCharacterById(characterId),
    [characterId],
  );

  useEffect(() => {
    const onSpeechEnd = (transcript: string) => {
      if (transcript && characterId) {
        resumeAudio();
        pushUserMessage(transcript);
        const latestHistory = useVoiceConversation.getState().history;
        const latestCharacter = getCharacterById(characterId);
        const messages = [
          {
            role: "system" as const,
            content: latestCharacter?.prompt || "You are a helpful assistant.",
          },
          ...latestHistory,
        ];
        start({ characterId, messages });
        resetTranscript();
      }
    };

    initVad(onSpeechEnd);
    return () => {
      const { disconnect: disconnectVad } = useVadStore.getState();
      disconnectVad();
      const { disconnect: disconnectLLM } = useVoiceConversation.getState();
      disconnectLLM();
    };
  }, [
    initVad,
    characterId,
    pushUserMessage,
    start,
    resetTranscript,
    resumeAudio,
  ]);

  useEffect(() => {
    if (isVadReady && !isConversationStarted && characterId) {
      resumeAudio();
      connectLLM(characterId);
      setIsConversationStarted(true);
    }
  }, [isVadReady, isConversationStarted, characterId, resumeAudio, connectLLM]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  });

  useEffect(() => {
    if (voiceActivity === VoiceActivity.Speaking && isPlaying) {
      interrupt();
    }
  }, [voiceActivity, interrupt, isPlaying]);

  const isUiReady = useMemo(() => {
    return isVadReady && isConversationStarted;
  }, [isVadReady, isConversationStarted]);

  const t = useTranslations("home");

  const animationActivity = useMemo(() => {
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
  }, [isVadReady, isConversationStarted, voiceActivity, isPlaying]);

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
                    if (characterId) {
                      const character = getCharacterById(characterId);
                      const messages = [
                        {
                          role: "system" as const,
                          content: character?.prompt ||
                            "You are a helpful assistant.",
                        },
                        ...updatedHistory,
                      ];
                      start({ characterId, messages });
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
                        src={currentCharacter.image}
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
                            defaultValue={msg.content}
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
                            {msg.content}
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

          <div
            className={cn(
              "absolute bottom-10 left-1/2 -translate-x-1/2 w-40 h-40 transition-opacity duration-300 ease-in-out",
              { "opacity-0": !isVadReady },
            )}
          >
            <AudioAnimation activity={animationActivity} />
          </div>
        </div>
      </div>
    </div>
  );
}
