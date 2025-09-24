"use client";
import { Alignment, Fit, Layout, type StateMachineInput, StateMachineInputType, useRive } from "@rive-app/react-canvas-lite";
import { useEffect, useRef, useState } from "react";
import { VoiceActivity } from "../types/vad";

interface AudioAnimationProps {
    activity: VoiceActivity;
}

export function AudioAnimation({ activity }: AudioAnimationProps) {
    const riveInputRef = useRef<StateMachineInput | null>(null);
    const { RiveComponent, rive } = useRive({
        src: "/ai_voice_states.riv",
        animations: ["listen", "speak", "think"],
        stateMachines: "StateMachine",
        layout: new Layout({ fit: Fit.Contain, alignment: Alignment.Center }),
        autoplay: true,
    });

    // Rive Number 输入：0=等待LLM返回语音 1=说话中(用户/LLM) 2=等待用户说话
    const [machineNumber, setMachineNumber] = useState<number>(0);

    useEffect(() => {
        // VoiceActivity 状态映射：
        // Loading: 等待LLM返回语音 -> 0
        // Speaking: 用户说话或LLM说话 -> 1  
        // Idle: 等待用户说话 -> 2
        switch (activity) {
            case VoiceActivity.Loading:
                setMachineNumber(0); // 等待LLM返回语音
                break;
            case VoiceActivity.Speaking:
                setMachineNumber(1); // 用户说话或LLM说话
                break;
            case VoiceActivity.Idle:
                setMachineNumber(2); // 等待用户说话
                break;
        }
    }, [activity]);

    useEffect(() => {
        if (!rive || !riveInputRef) return;
        if (!riveInputRef.current) {
            const inputs = rive.stateMachineInputs("StateMachine");
            if (!inputs) return;
            const numberInput = inputs.find(i => i.type === StateMachineInputType.Number);
            if (!numberInput) return;
            riveInputRef.current = numberInput;
        }
        riveInputRef.current.value = machineNumber;
    }, [machineNumber, rive]);

    return <RiveComponent />;
}
