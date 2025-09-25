"use client";
import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";

import { useVadStore } from "@/store/vad";
import { VoiceActivity } from "@/types/vad";
import { useVoiceConversation } from "@/store/voice-conversation";
import { getCharacterById } from "@/lib/characters";

import {
  Message,
  MessageAvatar,
  MessageContent,
} from "@/components/ui/message";
import { AudioAnimation } from "@/components/AudioAnimation";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { Loader } from "@/components/ui/loader";

export default function Page() {
  const params = useParams<{ id: string }>();
  const characterId = params?.id ?? "";

  const [isConversationStarted, setIsConversationStarted] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const { isVadReady, voiceActivity, transcript, initVad, resetTranscript } =
    useVadStore();
  // 不需要的值不解构以避免未使用变量警告
  const {
    history,
    start,
    pushUserMessage,
    isPlaying,
    resumeAudio,
    connect: connectLLM,
    interrupt,
    connection,
  } = useVoiceConversation();

  // 获取当前角色信息
  const currentCharacter = useMemo(
    () => getCharacterById(characterId),
    [characterId],
  );

  useEffect(() => {
    const onSpeechEnd = (transcript: string) => {
      if (transcript && characterId) {
        resumeAudio();
        pushUserMessage(transcript);
        start({
          characterId,
          messages: [
            { role: "system", content: "You are a helpful assistant." },
            { role: "user", content: transcript },
          ],
        });
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

  // biome-ignore lint/correctness/useExhaustiveDependencies: 需要实时监听自动滚动
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [history, transcript]);

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
    // After the user speaks, VAD enters a 'Loading' state.
    // This state persists until the next speech event.
    // After the AI finishes speaking, we should display the 'Idle' state
    // to show that the app is ready for user input, instead of the stale 'Loading' state.
    if (voiceActivity === VoiceActivity.Loading) {
      return VoiceActivity.Idle;
    }
    return voiceActivity;
  }, [isVadReady, isConversationStarted, voiceActivity, isPlaying]);

  // 连接状态指示器颜色
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
    <div className=" bg-gradient-to-br from-slate-50 to-slate-200 dark:from-slate-900 dark:to-slate-800  flex items-center justify-center min-h-screen w-full border-none">
      <div className="w-full h-[100dvh] flex flex-col justify-center items-center relative">
        {/* 角色信息头部 */}
        {currentCharacter && (
          <div className="absolute top-0 left-0 right-0 z-10 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm border-b border-slate-200 dark:border-slate-700">
            <div className="flex items-center justify-center py-3 px-4">
              <div className="flex items-center space-x-3">
                {/* 角色名称和连接状态 */}
                <div className="flex items-center space-x-2">
                  <h1 className="text-lg font-semibold text-slate-800 dark:text-slate-200">
                    {currentCharacter.name}
                  </h1>
                  {/* 连接状态小圆点 */}
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
            "w-full flex-1 overflow-y-auto pb-40 no-scrollbar [mask-image:linear-gradient(to_bottom,black_calc(100%-10rem),transparent)] transition-all duration-300 ease-in-out pointer-events-auto",
            currentCharacter ? "pt-20" : "pt-4", // 如果有角色信息头部，增加顶部padding
            { "opacity-60 blur-sm": !isUiReady },
          )}
        >
          <div className="px-4 space-y-4 mx-auto w-[80vw]">
            {history.map((msg, index) => (
              <Message
                key={`${msg.role}-${index}`}
                className={cn("items-start gap-4", {
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
                <MessageContent
                  className={cn(
                    { "bg-white": msg.role === "user" },
                    { "bg-transparent p-0": msg.role === "assistant" },
                  )}
                >
                  {msg.content}
                </MessageContent>
              </Message>
            ))}
            {voiceActivity === VoiceActivity.Speaking && transcript && (
              <Message className="items-start justify-end gap-4">
                <MessageContent className="bg-white">
                  {transcript}
                </MessageContent>
              </Message>
            )}
          </div>
        </div>

        <div className="absolute inset-0 pointer-events-none">
          {!isVadReady && (
            <div className="absolute inset-0 flex justify-center items-center">
              <Loader />
              {/* Optional text near loader for accessibility */}
              <span className="sr-only">{t("vad_loading")}</span>
            </div>
          )}
          {/* When VAD is ready and conversation started, show prompt based on voice activity */}
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
              { "opacity-0": !isVadReady }, // Hide animation until VAD is ready
            )}
          >
            <AudioAnimation activity={animationActivity} />
          </div>
        </div>
      </div>
    </div>
  );
}
