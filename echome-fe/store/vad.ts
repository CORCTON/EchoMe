import { create } from "zustand";
import { VoiceActivity } from "@/types/vad";
import { MicVAD } from "@ricky0123/vad-web";

export enum ConnectionState {
    Connecting,
    Connected,
    Disconnected,
}

interface VadState {
    connectionState: ConnectionState;
    voiceActivity: VoiceActivity;
    transcript: string;
    socket: WebSocket | null;
    vad: MicVAD | null;
    initVad: () => Promise<void>;
    connect: () => void;
    disconnect: () => void;
    send: (data: Float32Array) => void;
    setVoiceActivity: (activity: VoiceActivity) => void;
}

export const useVadStore = create<VadState>((set, get) => ({
    connectionState: ConnectionState.Disconnected,
    voiceActivity: VoiceActivity.Loading,
    vad: null,
    initVad: async () => {
        set({ voiceActivity: VoiceActivity.Loading });
        try {
            const vad = await MicVAD.new({
                baseAssetPath: "/vad/",
                onnxWASMBasePath: "/vad/",
                model: "v5",
                preSpeechPadMs: 200,
                positiveSpeechThreshold: 0.8,
                minSpeechMs: 100,
                onSpeechStart: () => {
                    set({ voiceActivity: VoiceActivity.Speaking });
                },
                onSpeechEnd: () => {
                    set({ voiceActivity: VoiceActivity.Idle });
                },
                onFrameProcessed: (probabilities, frame) => {
                    if (probabilities.isSpeech > 0.8) {
                        get().send(frame);
                    }
                },
            });
            set({ vad, voiceActivity: VoiceActivity.Idle });
            vad.start();
        } catch (e) {
            console.error("Failed to initialize VAD", e);
            set({ voiceActivity: VoiceActivity.Idle });
        }
    },
    transcript: "",
    socket: null,
    connect: () => {
        const existingSocket = get().socket;
        if (existingSocket && existingSocket.readyState < 2) { // CONNECTING or OPEN
            return;
        }

        set({ connectionState: ConnectionState.Connecting });

        const url = new URL(`${process.env.NEXT_PUBLIC_WS_URL}/ws/asr`);
        url.searchParams.set("sample_rate", "16000");
        url.searchParams.set("format", "pcm");
        const newSocket = new WebSocket(url);

        newSocket.onopen = () => {
            set({ connectionState: ConnectionState.Connected });
            console.log("WebSocket connected");
        };

        newSocket.onmessage = (event) => {
            const message = JSON.parse(event.data);
            // The backend sends { "type": "asr_result", "text": "...", "sentence_end": false }
            if (message.type === "asr_result" && message.text) {
                // Replace the transcript with the new incoming text for a real-time correction effect
                set({ transcript: message.text });
            }
        };

        newSocket.onclose = () => {
            set((state) => (state.socket === newSocket ? { connectionState: ConnectionState.Disconnected, socket: null } : {}));
            console.log("WebSocket disconnected");
        };

        newSocket.onerror = (error) => {
            console.error("WebSocket error:", error);
            newSocket.close();
        };

        set({ socket: newSocket });
    },
    disconnect: () => {
        const { socket, vad } = get();
        if (socket) {
            // Remove listeners to prevent memory leaks and unwanted state updates
            socket.onopen = null;
            socket.onmessage = null;
            socket.onclose = null;
            socket.onerror = null;
            socket.close();
        }
        vad?.destroy();
        set({ socket: null, connectionState: ConnectionState.Disconnected, vad: null });
    },
    send: (data: Float32Array) => {
        const socket = get().socket;
        if (socket && socket.readyState === WebSocket.OPEN) {
            const buffer = new Int16Array(data.length);
            for (let i = 0; i < data.length; i++) {
                buffer[i] = Math.max(-32768, Math.min(32767, Math.floor(data[i] * 32768)));
            }
            socket.send(buffer.buffer);
        }
    },
    setVoiceActivity: (activity: VoiceActivity) => {
        // The transcript is now cleared by the arrival of the first result of the next sentence,
        // which prevents a race condition where the final result of the previous sentence is cleared
        // before it can be displayed.
        set({ voiceActivity: activity });
    },
}));
