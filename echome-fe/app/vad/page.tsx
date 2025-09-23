"use client";
import { Alignment, Fit, Layout, type StateMachineInput, StateMachineInputType, useRive } from "@rive-app/react-canvas-lite";
import { useEffect, useRef, useState } from "react";
import { useVoiceActivity, VoiceActivity } from "@/hooks/useVoiceActivity";

export default function Page() {
    const riveInputRef = useRef<StateMachineInput | null>(null);

    const { activity } = useVoiceActivity({
        onChunk: (chunk) => {
            // TODO: 发送到后端（流式）
            console.log("chunk", chunk.size, chunk.type, Date.now());
        }
    });

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

    return (
        <div className="bg-gray-200 flex items-center justify-center min-h-screen">
            <div className="w-full max-w-3xl h-screen">
                <RiveComponent />
            </div>
        </div>
    );
}
