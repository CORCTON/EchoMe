import { create } from "zustand";
import { VoiceActivity } from "@/types/vad";
import { MicVAD } from "@ricky0123/vad-web";
import { parseApiResponse } from "@/lib/utils";

export enum ConnectionState {
    Connecting,
    Connected,
    Disconnected,
}

interface AsrResult {
    type: "asr_result";
    text: string;
    sentence_end: boolean;
}

interface VadState {
    isVadReady: boolean;
    asrConnectionState: ConnectionState;
    voiceActivity: VoiceActivity;
    transcript: string;
    isFinal: boolean;
    isTranscribing: boolean;
    socket: WebSocket | null;
    vad: MicVAD | null;
    initVad: () => Promise<void>;
    disconnect: () => void;
    send: (data: Float32Array) => void;
    setVoiceActivity: (activity: VoiceActivity) => void;
    resetTranscript: () => void;
}

export const useVadStore = create<VadState>((set, get) => ({
    isVadReady: false,
    asrConnectionState: ConnectionState.Disconnected,
    voiceActivity: VoiceActivity.Loading,
    isTranscribing: false,
    vad: null,
    initVad: async () => {
        set({ voiceActivity: VoiceActivity.Loading, isVadReady: false });
        try {
            const vad = await MicVAD.new({
                baseAssetPath: "/vad/",
                onnxWASMBasePath: "/vad/",
                model: "v5",
                preSpeechPadMs: 200,
                positiveSpeechThreshold: 0.8,
                minSpeechMs: 100,
                onSpeechStart: () => {
                    get().resetTranscript();
                    set({ voiceActivity: VoiceActivity.Speaking, isTranscribing: true });
                },
                onSpeechEnd: () => {
                    set({ voiceActivity: VoiceActivity.Loading, isTranscribing: false });
                    const { socket } = get();
                    if (socket) {
                        socket.close();
                        set({ socket: null, asrConnectionState: ConnectionState.Disconnected });
                    }
                },
                onFrameProcessed: (_probabilities, frame) => {
                    if (get().isTranscribing) {
                        get().send(frame);
                    }
                },
            });
            set({ vad, voiceActivity: VoiceActivity.Idle, isVadReady: true });
            vad.start();
        } catch (e) {
            console.error("Failed to initialize VAD", e);
            set({ voiceActivity: VoiceActivity.Idle, isVadReady: false });
        }
    },
    transcript: "",
    isFinal: false,
    socket: null,
    disconnect: () => {
        const { socket, vad } = get();
        if (socket) {
            socket.onopen = null;
            socket.onmessage = null;
            socket.onclose = null;
            socket.onerror = null;
            socket.close();
        }
        vad?.destroy();
        set({ socket: null, asrConnectionState: ConnectionState.Disconnected, vad: null, isVadReady: false, isTranscribing: false });
    },
    send: (data: Float32Array) => {
        let { socket } = get();
        if (!socket || socket.readyState > 1) {
            set({ asrConnectionState: ConnectionState.Connecting });
            const url = new URL(`${process.env.NEXT_PUBLIC_WS_URL}/ws/asr`);
            url.searchParams.set("sample_rate", "16000");
            url.searchParams.set("format", "pcm");
            socket = new WebSocket(url);

            socket.onopen = () => {
                set({ asrConnectionState: ConnectionState.Connected, socket });
                const buffer = new Int16Array(data.length);
                for (let i = 0; i < data.length; i++) {
                    buffer[i] = Math.max(-32768, Math.min(32767, Math.floor(data[i] * 32768)));
                }
                if (socket) {
                    socket.send(buffer.buffer);
                }
            };

            socket.onmessage = (event) => {
                if (typeof event.data !== "string") return;
                const result = parseApiResponse<AsrResult>(event.data);
                if (!result.ok || !result.value.success || !result.value.data) {
                    return;
                }
                const message = result.value.data;
                if (message?.type === "asr_result") {
                    const text = typeof message.text === "string" ? message.text : "";
                    const isFinal = Boolean(message.sentence_end);
                    set({ transcript: text, isFinal });
                }
            };

            socket.onclose = () => {
                set((state) => (state.socket === socket ? { asrConnectionState: ConnectionState.Disconnected, socket: null } : {}));
            };

            socket.onerror = (error) => {
                console.error("ASR WebSocket error:", error);
                const { socket: currentSocket } = get();
                if (currentSocket) currentSocket.close();
            };
            set({ socket });
        } else if (socket && socket.readyState === WebSocket.OPEN) {
            const buffer = new Int16Array(data.length);
            for (let i = 0; i < data.length; i++) {
                buffer[i] = Math.max(-32768, Math.min(32767, Math.floor(data[i] * 32768)));
            }
            socket.send(buffer.buffer);
        }
    },
    setVoiceActivity: (activity: VoiceActivity) => {
        set({ voiceActivity: activity });
    },
    resetTranscript: () => set({ transcript: "", isFinal: false }),
}));
