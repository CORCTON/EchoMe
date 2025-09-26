import { create } from "zustand";
import { VoiceActivity } from "@/types/vad";
import { MicVAD } from "@ricky0123/vad-web";

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
  committedTranscript: string;
  transcript: string;
  isFinal: boolean;
  isTranscribing: boolean;
  socket: WebSocket | null;
  vad: MicVAD | null;
  preSpeechBuffer: Float32Array[];
  audioQueue: Float32Array[];
  initVad: (onSpeechEnd: (transcript: string) => void) => Promise<void>;
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
  preSpeechBuffer: [],
  audioQueue: [],
  committedTranscript: "",
  initVad: async (onSpeechEndCallback) => {
    set({ voiceActivity: VoiceActivity.Loading, isVadReady: false });
    try {
      const vad = await MicVAD.new({
        baseAssetPath: "/vad/",
        onnxWASMBasePath: "/vad/",
        model: "v5",
        preSpeechPadMs: 200,
        positiveSpeechThreshold: 0.9,
        minSpeechMs: 100,
        onSpeechStart: () => {
          const { echoGuardUntil, interrupt } =
            require("@/store/voice-conversation").useVoiceConversation.getState();
          if (echoGuardUntil && Date.now() < echoGuardUntil) {
            return;
          }
          interrupt();
          get().resetTranscript();
          const { preSpeechBuffer, send } = get();
          if (preSpeechBuffer.length > 0) {
            const concatenated = new Float32Array(
              preSpeechBuffer.reduce((acc, val) => acc + val.length, 0),
            );
            let offset = 0;
            for (const chunk of preSpeechBuffer) {
              concatenated.set(chunk, offset);
              offset += chunk.length;
            }
            send(concatenated);
          }
          set({
            voiceActivity: VoiceActivity.Speaking,
            isTranscribing: true,
            preSpeechBuffer: [],
          });
        },
        onSpeechEnd: () => {
          set({
            voiceActivity: VoiceActivity.Loading,
            isTranscribing: false,
            preSpeechBuffer: [],
          });
          const { socket, transcript } = get();
          if (socket) {
            socket.close();
            set({
              socket: null,
              asrConnectionState: ConnectionState.Disconnected,
            });
          }
          if (transcript) {
            onSpeechEndCallback(transcript);
          }
        },
        onFrameProcessed: (_, frame) => {
          const { isTranscribing, preSpeechBuffer, send } = get();
          if (isTranscribing) {
            send(frame);
          } else {
            const bufferSize = 7;
            const newBuffer = [...preSpeechBuffer, frame];
            if (newBuffer.length > bufferSize) {
              newBuffer.shift();
            }
            set({ preSpeechBuffer: newBuffer });
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
    set({
      socket: null,
      asrConnectionState: ConnectionState.Disconnected,
      vad: null,
      isVadReady: false,
      isTranscribing: false,
    });
  },
  send: (data: Float32Array) => {
    const { socket } = get();

    const sendData = (ws: WebSocket, audioData: Float32Array) => {
      const buffer = new Int16Array(audioData.length);
      for (let i = 0; i < audioData.length; i++) {
        buffer[i] = Math.max(
          -32768,
          Math.min(32767, Math.floor(audioData[i] * 32768)),
        );
      }
      ws.send(buffer.buffer);
    };

    if (
      !socket ||
      socket.readyState === WebSocket.CLOSED ||
      socket.readyState === WebSocket.CLOSING
    ) {
      set({ asrConnectionState: ConnectionState.Connecting, audioQueue: [] });
      const newSocket = new WebSocket(
        `${process.env.NEXT_PUBLIC_WS_URL}/ws/asr?sample_rate=16000&format=pcm`,
      );

      newSocket.onopen = () => {
        set({
          asrConnectionState: ConnectionState.Connected,
          socket: newSocket,
        });
        sendData(newSocket, data); // Send the initial data (pre-speech buffer)
        const { audioQueue } = get();
        for (const queuedData of audioQueue) {
          sendData(newSocket, queuedData);
        }
        set({ audioQueue: [] });
      };

      newSocket.onmessage = (event) => {
        if (typeof event.data !== "string") return;
        try {
          const message = JSON.parse(event.data) as AsrResult;
          if (message?.type === "asr_result") {
            const text = message.text || "";
            const isFinal = !!message.sentence_end;

            set((state) => {
              const newTranscript = state.committedTranscript + text;
              let newCommittedTranscript = state.committedTranscript;
              if (isFinal) {
                newCommittedTranscript = newTranscript
                  ? `${newTranscript} `
                  : state.committedTranscript;
              }
              return {
                transcript: newTranscript,
                isFinal,
                committedTranscript: newCommittedTranscript,
              };
            });
          }
        } catch (e) {
          console.error("Failed to parse asr message", e);
        }
      };

      newSocket.onclose = () => {
        set((state) =>
          state.socket === newSocket
            ? { asrConnectionState: ConnectionState.Disconnected, socket: null }
            : {},
        );
      };

      newSocket.onerror = (error) => {
        console.error("ASR WebSocket error:", error);
        get().disconnect();
      };

      set({ socket: newSocket });
    } else if (socket.readyState === WebSocket.CONNECTING) {
      set((state) => ({ audioQueue: [...state.audioQueue, data] }));
    } else if (socket.readyState === WebSocket.OPEN) {
      sendData(socket, data);
    }
  },
  setVoiceActivity: (activity: VoiceActivity) => {
    set({ voiceActivity: activity });
  },
  resetTranscript: () =>
    set({ transcript: "", committedTranscript: "", isFinal: false }),
}));
