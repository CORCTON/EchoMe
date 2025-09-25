"use client";
import { create } from "zustand";

type Role = "system" | "user" | "assistant";

export type ChatMessage = {
  role: Role;
  content: string;
};

export type ConversationRequest = {
  characterId: string;
  messages: ChatMessage[];
};

type ConnectionState = "idle" | "connecting" | "connected" | "disconnected";

export interface VoiceConversationState {
  connection: ConnectionState;
  ws: WebSocket | null;
  characterId: string | null;
  history: ChatMessage[];
  audioCtx: AudioContext | null;
  gainNode: GainNode | null;
  isPlaying: boolean;
  sources: AudioBufferSourceNode[];
  nextStartTime?: number;
  idleTimer: NodeJS.Timeout | null;
  reconnectTimer: NodeJS.Timeout | null;
  isInterrupted: boolean;
  echoGuardUntil: number | null;
  connect: (characterId: string) => void;
  start: (payload: ConversationRequest) => void;
  pushUserMessage: (content: string) => void;
  interrupt: () => void;
  disconnect: () => void;
  stopPlaying: () => void;
  clear: () => void;
  resumeAudio: () => void;
}

function int16ToAudioBuffer(
  ctx: AudioContext,
  int16: Int16Array,
  sampleRate = 24000,
) {
  const float32 = new Float32Array(int16.length);
  for (let i = 0; i < int16.length; i++) {
    float32[i] = Math.max(-1, Math.min(1, int16[i] / 32768));
  }
  const buffer = ctx.createBuffer(1, float32.length, sampleRate);
  buffer.copyToChannel(float32, 0);
  return buffer;
}

export const useVoiceConversation = create<VoiceConversationState>(
  (set, get) => ({
    connection: "idle",
    ws: null,
    characterId: null,
    history: [],
    audioCtx: null,
    gainNode: null,
    isPlaying: false,
    sources: [],
    nextStartTime: undefined,
    idleTimer: null,
    reconnectTimer: null,
    isInterrupted: false,
    echoGuardUntil: null,

    connect: (characterId: string) => {
      const { ws, reconnectTimer } = get();
      if (ws && ws.readyState < 2) return; // Already connected or connecting
      if (reconnectTimer) clearTimeout(reconnectTimer);

      set({ connection: "connecting", characterId, reconnectTimer: null });

      const url = new URL(
        `${process.env.NEXT_PUBLIC_WS_URL}/ws/voice-conversation`,
      );
      const newWs = new WebSocket(url);

      newWs.binaryType = "arraybuffer";

      newWs.onopen = () => {
        set({ connection: "connected", ws: newWs });
      };

      newWs.onmessage = async (ev) => {
        if (get().isInterrupted) return;

        const { idleTimer } = get();
        if (idleTimer) clearTimeout(idleTimer);

        if (typeof ev.data === "string") {
          try {
            const message = JSON.parse(ev.data);

            if (message.type === "stream_chunk" && message.content) {
              set((s) => {
                const history = [...s.history];
                const last = history[history.length - 1];
                if (last?.role === "assistant") {
                  history[history.length - 1] = {
                    role: "assistant",
                    content: (last.content ?? "") + message.content,
                  };
                } else {
                  history.push({ role: "assistant", content: message.content });
                }
                return { history };
              });
            } else if (message.type === "text_response" && message.response) {
              set((s) => {
                const history = [...s.history];
                const last = history[history.length - 1];
                if (last?.role === "assistant") {
                  history[history.length - 1] = {
                    role: "assistant",
                    content: message.response,
                  };
                } else {
                  history.push({
                    role: "assistant",
                    content: message.response,
                  });
                }
                return { history };
              });
            } else if (message.type === "tts_error") {
              console.warn("TTS error ignored:", message.message);
            }
          } catch (e) {
            console.error("Failed to parse ws message", e);
          }
          return;
        }

        const arrayBuf = ev.data as ArrayBuffer;
        const int16 = new Int16Array(arrayBuf);
        const { audioCtx, gainNode, nextStartTime, isPlaying } = get();
        if (!audioCtx || !gainNode) return;

        try {
          const audioBuffer = int16ToAudioBuffer(audioCtx, int16);
          const source = audioCtx.createBufferSource();
          source.buffer = audioBuffer;
          source.connect(gainNode);

          const now = audioCtx.currentTime;
          const startAt =
            nextStartTime && nextStartTime > now ? nextStartTime : now;

          if (!isPlaying) {
            set({ echoGuardUntil: Date.now() + 300 });
          }

          source.start(startAt);
          const endTime = startAt + audioBuffer.duration;

          source.onended = () => {
            set((s) => ({
              sources: s.sources.filter((src) => src !== source),
            }));

            const state = get();
            if (state.sources.length === 0) {
              const idleTimer = setTimeout(() => {
                const latest = get();
                if (latest.sources.length === 0) {
                  set({
                    isPlaying: false,
                    nextStartTime: undefined,
                    idleTimer: null,
                  });
                }
              }, 350);
              set({ idleTimer });
            }
          };

          set((s) => ({
            isPlaying: true,
            nextStartTime: endTime,
            sources: [...s.sources, source],
          }));
        } catch (err) {
          console.error("Failed to play pcm chunk", err);
        }
      };

      newWs.onerror = (e) => {
        console.error("voice-conversation ws error", e);
        // onclose will be called next, which handles reconnection.
      };

      newWs.onclose = () => {
        set({ connection: "disconnected", ws: null });
        const { characterId: currentCharacterId } = get();
        if (currentCharacterId) {
          console.log(
            "WebSocket connection lost, attempting to reconnect in 2s...",
          );
          const timer = setTimeout(() => {
            get().connect(currentCharacterId);
          }, 2000);
          set({ reconnectTimer: timer });
        }
      };
    },

    start: (payload: ConversationRequest) => {
      const { ws, connection } = get();
      if (ws && connection === "connected") {
        set({ isInterrupted: false });
        ws.send(JSON.stringify({ ...payload, stream: true }));
      } else {
        console.error("WebSocket not connected. Cannot start conversation.");
      }
    },

    pushUserMessage: (content: string) => {
      if (!content) return;
      set((s) => ({ history: [...s.history, { role: "user", content }] }));
    },

    stopPlaying: () => {
      const { sources, idleTimer } = get();
      if (idleTimer) clearTimeout(idleTimer);

      sources.forEach((source) => {
        try {
          source.stop();
        } catch {
          // Ignore errors if source has already stopped
        }
      });

      set({
        isPlaying: false,
        sources: [],
        idleTimer: null,
      });
    },

    interrupt: () => {
      get().stopPlaying();
      set({ isInterrupted: true });
    },

    disconnect: () => {
      const { ws, audioCtx, idleTimer, reconnectTimer } = get();
      if (reconnectTimer) clearTimeout(reconnectTimer);
      set({ characterId: null, reconnectTimer: null }); // Prevent reconnection

      if (ws && ws.readyState < 2) {
        try {
          ws.close();
        } catch {}
      }
      if (audioCtx) {
        try {
          audioCtx.close();
        } catch {}
      }
      if (idleTimer) clearTimeout(idleTimer);
      set({
        ws: null,
        audioCtx: null,
        gainNode: null,
        isPlaying: false,
        connection: "idle",
        idleTimer: null,
        history: [],
      });
    },

    clear: () => set({ history: [] }),

    resumeAudio: () => {
      let { audioCtx } = get();
      if (!audioCtx) {
        try {
          let Ctx: typeof AudioContext | undefined = window.AudioContext;
          const w = window as unknown as {
            webkitAudioContext?: typeof AudioContext;
          };
          if (!Ctx && w.webkitAudioContext) Ctx = w.webkitAudioContext;
          if (Ctx) {
            audioCtx = new Ctx();
            const gain = audioCtx.createGain();
            gain.gain.value = 1.0;
            gain.connect(audioCtx.destination);
            set({ audioCtx, gainNode: gain });
          } else {
            console.error("AudioContext not supported in this environment");
            return;
          }
        } catch (e) {
          console.error("Failed to create AudioContext", e);
          return;
        }
      }

      if (audioCtx.state === "suspended") {
        audioCtx.resume();
      }
    },
  }),
);
