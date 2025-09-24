"use client";
import { useEffect, useMemo } from "react";
import { useVoiceActivity } from "../../hooks/useVoiceActivity";

import { ConnectionState, useVadStore } from "@/store/vad";
import { VoiceActivity } from "@/types/vad";

import { Message, MessageContent, MessageAvatar } from "@/components/ui/message";
import { AudioAnimation } from "@/components/AudioAnimation";
import { cn } from "@/lib/utils";

export default function Page() {
    const { connectionState, voiceActivity, transcript, connect, send } = useVadStore();

    const messages = [
        { role: "user", text: transcript },
        { role: "assistant", text: "(AI 回复，后端未实现)" },
    ];

    useVoiceActivity({
        onFrameProcessed: (frame: Float32Array) => {
            send(frame);
        }
    });

    useEffect(() => {
        connect();
        return () => {
            const { disconnect } = useVadStore.getState();
            disconnect();
        };
    }, [connect]);

    const isReady = useMemo(() => {
        return connectionState === ConnectionState.Connected && voiceActivity !== VoiceActivity.Loading;
    }, [connectionState, voiceActivity]);

    return (
        <div className="bg-gray-200 flex items-center justify-center min-h-screen overflow-hidden">
            <div className="w-full max-w-3xl h-screen flex flex-col justify-center items-center relative">
                {/* Messages list - only render messages[0] for now using shared Message components */}
                {isReady && (
                    <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-full mb-4 w-full max-w-xl px-4 transition-all duration-500">
                        <div className="flex flex-col gap-4">
                            {/* 用户输入，靠右 */}
                            {messages[0].text && (
                                <Message className="justify-end">
                                    <MessageContent>
                                        {messages[0].text}
                                    </MessageContent>
                                </Message>
                            )}

                            {/* AI回复，靠左 */}
                            {messages[1].text && (
                                <Message className="justify-start">
                                    <MessageAvatar src="/avatars/ai.png" alt="AI" fallback="AI" />
                                    <MessageContent className="bg-transparent p-0">
                                        {messages[1].text}
                                    </MessageContent>
                                </Message>
                            )}
                        </div>
                    </div>
                )}

                {/* AudioRive动画 */}
                <div
                    className={cn(
                        "transition-all duration-300 ease-in-out",
                        {
                            "absolute bottom-10 w-40 h-40": isReady,
                            "w-full h-full": !isReady,
                        }
                    )}
                >
                    <AudioAnimation activity={voiceActivity} />
                </div>
            </div>
        </div>
    );
}
