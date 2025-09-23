"use client";
import { Alignment, Fit, Layout, type StateMachineInput, StateMachineInputType, useRive } from "@rive-app/react-canvas-lite";
import { useEffect, useRef, useState } from "react";
import { VoiceActivity } from "../../hooks/useVoiceActivity";

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

    // Rive Number 输入：0=loading 1=说话中 2=空闲/思考
    const [machineNumber, setMachineNumber] = useState<number>(0);

    useEffect(() => {
        // 语音活动（VAD）和 Rive 显示状态解耦：
        // 说话结束时应立即进入 Idle（2），不应等待播放器完成播放才显示 Idle。
        switch (activity) {
            case VoiceActivity.Loading:
                setMachineNumber(0);
                break;
            case VoiceActivity.Speaking:
                setMachineNumber(1);
                break;
            case VoiceActivity.Idle:
                setMachineNumber(2);
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
