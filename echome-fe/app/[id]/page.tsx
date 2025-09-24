"use client";
import { useEffect, useMemo } from "react";

import { ConnectionState, useVadStore } from "@/store/vad";
import { VoiceActivity } from "@/types/vad";

import {
    Message,
    MessageAvatar,
    MessageContent,
} from "@/components/ui/message";
import { AudioAnimation } from "@/components/AudioAnimation";
import { cn } from "@/lib/utils";
import { Loader } from "@/components/ui/loader";

export default function Page() {
    const { connectionState, voiceActivity, transcript, connect, initVad } =
        useVadStore();

    const messages = [
        { role: "user", text: transcript },
        // { role: "assistant", text: "(AI 回复，未实现)" },
    ];

    useEffect(() => {
        connect();
        initVad();
        return () => {
            const { disconnect } = useVadStore.getState();
            disconnect();
        };
    }, [connect, initVad]);

    const isReady = useMemo(() => {
        return connectionState === ConnectionState.Connected &&
            voiceActivity !== VoiceActivity.Loading;
    }, [connectionState, voiceActivity]);

    return (
        <div className="bg-gray-200 flex items-center justify-center min-h-screen overflow-hidden">
            <div className="w-full  h-screen flex flex-col justify-center items-center relative">
                {/* 消息列表 */}
                <div
                    className={cn(
                        "w-full flex-1 overflow-y-auto pt-4 pb-40 no-scrollbar [mask-image:linear-gradient(to_bottom,black_calc(100%-10rem),transparent)] transition-all duration-300 ease-in-out",
                        { "opacity-60 blur-sm pointer-events-none": !isReady },
                    )}
                >
                    <div className="px-4 space-y-4 mx-auto w-[80vw]">
                        {messages.map((msg, index) => (
                            msg.text && (
                                <Message
                                    key={`${msg.role}-${index}`}
                                    className={cn("items-start gap-4", {
                                        "justify-end": msg.role === "user",
                                        "justify-start":
                                            msg.role === "assistant",
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
                                            {
                                                "bg-transparent p-0":
                                                    msg.role === "assistant",
                                            },
                                        )}
                                    >
                                        {msg.text}
                                    </MessageContent>
                                </Message>
                            )
                        ))}
                    </div>
                </div>

                {/* 动画容器 */}
                <div className="absolute inset-0 pointer-events-none">
                    {/* 准备Loader*/}
                    <div
                        className={cn(
                            "absolute inset-0 transition-opacity duration-300 ease-in-out",
                            { "opacity-0": isReady },
                        )}
                    >
                        <Loader />
                    </div>

                    {/* Ready Rive动画 */}
                    {isReady && (
                        <div
                            className={cn(
                                "absolute bottom-10 left-1/2 -translate-x-1/2 w-40 h-40 transition-opacity duration-300 ease-in-out",
                            )}
                        >
                            <AudioAnimation activity={voiceActivity} />
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
