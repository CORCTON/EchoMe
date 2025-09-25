"use client";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";

import { useVadStore } from "@/store/vad";
import { VoiceActivity } from "@/types/vad";
import { useVoiceConversation } from "@/store/voice-conversation";

import {
    Message,
    MessageAvatar,
    MessageContent,
} from "@/components/ui/message";
import { AudioAnimation } from "@/components/AudioAnimation";
import { cn } from "@/lib/utils";
import { Loader } from "@/components/ui/loader";

export default function Page() {
    const params = useParams<{ id: string }>();
    const characterId = params?.id ?? "";

    const [isConversationStarted, setIsConversationStarted] = useState(false);

    const { isVadReady, voiceActivity, transcript, isFinal, initVad, resetTranscript } =
        useVadStore();
    const { history, start, pushUserMessage, interrupt, connection: llmConnection, isPlaying, resumeAudio, connect: connectLLM } = useVoiceConversation();

    useEffect(() => {
        initVad();
        return () => {
            const { disconnect } = useVadStore.getState();
            disconnect();
            const { interrupt: interruptLLM } = useVoiceConversation.getState();
            interruptLLM();
        };
    }, [initVad]);

    const handleStartConversation = () => {
        if (isConversationStarted || !isVadReady) return;
        resumeAudio();
        connectLLM(characterId);
        setIsConversationStarted(true);
    };

    useEffect(() => {
        if (voiceActivity === VoiceActivity.Speaking && (llmConnection === "connected" || isPlaying)) {
            interrupt();
        }
    }, [voiceActivity, interrupt, llmConnection, isPlaying]);

    useEffect(() => {
        if (isConversationStarted && isFinal && transcript && characterId) {
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
    }, [isConversationStarted, isFinal, transcript, characterId, pushUserMessage, start, resetTranscript]);

    const isUiReady = useMemo(() => {
        return isVadReady && isConversationStarted;
    }, [isVadReady, isConversationStarted]);

    const animationActivity = useMemo(() => {
        if (!isVadReady || !isConversationStarted) {
            return VoiceActivity.Loading;
        }
        return voiceActivity;
    }, [isVadReady, isConversationStarted, voiceActivity]);

    return (
        <button
            type="button"
            className=" bg-gradient-to-br from-slate-50 to-slate-200 dark:from-slate-900 dark:to-slate-800  flex items-center justify-center min-h-screen overflow-hidden w-full border-none cursor-pointer"
            onClick={handleStartConversation}
            disabled={!isVadReady || isConversationStarted}
        >
            <div className="w-full h-[100dvh] flex flex-col justify-center items-center relative pointer-events-none">
                <div
                    className={cn(
                        "w-full flex-1 overflow-y-auto pt-4 pb-40 no-scrollbar [mask-image:linear-gradient(to_bottom,black_calc(100%-10rem),transparent)] transition-all duration-300 ease-in-out",
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
                                {msg.role === "assistant" && (
                                    <MessageAvatar
                                        src="/avatars/ai.png"
                                        alt="AI"
                                        fallback="AI"
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
                    </div>
                </div>

                <div className="absolute inset-0">
                    {!isVadReady && (
                        <div className="absolute inset-0 flex justify-center items-center">
                            <Loader />
                        </div>
                    )}
                    {isVadReady && !isConversationStarted && (
                        <div className="absolute inset-0 flex justify-center items-center">
                            <p className="text-lg text-gray-600">轻点屏幕以开始</p>
                        </div>
                    )}

                    <div
                        className={cn(
                            "absolute bottom-10 left-1/2 -translate-x-1/2 w-40 h-40 transition-opacity duration-300 ease-in-out",
                            { "opacity-0": !isVadReady } // Hide animation until VAD is ready
                        )}
                    >
                        <AudioAnimation activity={animationActivity} />
                    </div>
                </div>
            </div>
        </button>
    );
}
