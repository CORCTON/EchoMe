"use client";
import { Alignment, Fit, Layout, type StateMachineInput, StateMachineInputType, useRive } from "@rive-app/react-canvas-lite";
import { useEffect, useRef, useState } from "react";
import { useMicVAD } from "@ricky0123/vad-react";

// 预挑可用的 mimeType（按浏览器常见支持顺序）
const PICK_MIME_TYPES = ["audio/webm;codecs=opus", "audio/ogg;codecs=opus", "audio/webm", "audio/ogg"];

function pickSupportedMimeType(): string | undefined {
  return PICK_MIME_TYPES.find(MediaRecorder.isTypeSupported);
}

export default function Page() {
    const riveInputRef = useRef<StateMachineInput | null>(null);
        const mimeTypeRef = useRef<string | undefined>(undefined);
    const mediaStreamRef = useRef<MediaStream | null>(null); // 麦克风原始流
    const recorderRef = useRef<MediaRecorder | null>(null);  // 当前活动的 recorder（只在说话段存在）

    // 初始化麦克风流（组件挂载时）
    useEffect(() => {
        let cancelled = false;
        async function initStream() {
            try {
                const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
                if (cancelled) return;
                mediaStreamRef.current = stream;
                // 选择一次 mimeType
                mimeTypeRef.current = pickSupportedMimeType();
                if (!mimeTypeRef.current) {
                    console.warn("当前浏览器不支持 Opus 编码的 MediaRecorder");
                }
            } catch (e) {
                console.error("获取麦克风失败", e);
            }
        }
        initStream();
        return () => {
            cancelled = true;
            mediaStreamRef.current?.getTracks().forEach(t => { t.stop(); });
        };
    }, []);

    // 每段语音开始：启动 MediaRecorder（timeslice 发送 chunk）
    function startSegmentRecorder() {
        if (!mediaStreamRef.current) return;
        if (!mimeTypeRef.current) return; // 不支持就不启用
        // 避免重入
        if (recorderRef.current && recorderRef.current.state === "recording") return;
        const r = new MediaRecorder(mediaStreamRef.current, { mimeType: mimeTypeRef.current });
        r.ondataavailable = ev => {
            if (ev.data && ev.data.size > 0) {
                // TODO: 发送到后端（流式）
                console.log("chunk", ev.data.size, ev.data.type, Date.now());
            }
        };
        r.onstop = () => { recorderRef.current = null; setMachineNumber(2); };
        recorderRef.current = r;
        // 200ms 分片，可按需要调整（越小实时性越高，开销越大）
        r.start(200);
    }

    function stopSegmentRecorder() {
        const r = recorderRef.current;
        if (r && r.state === "recording") {
            r.stop();
        }
    }

    // VAD：语音开始/结束驱动 recorder 生命周期
    const vad = useMicVAD({
        model: "v5",
        baseAssetPath: "/vad/",
        onnxWASMBasePath: "/vad/",
        onSpeechRealStart: () => {
            setMachineNumber(1); // 说话中
            startSegmentRecorder();
        },
        onSpeechEnd() {
            stopSegmentRecorder();
        },
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
        setMachineNumber(vad.loading ? 0 : 2);
    }, [vad.loading]);

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
