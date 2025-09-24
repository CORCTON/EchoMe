"use client";
import { create } from "zustand";
import { VoiceActivity } from "@/types/vad";
import { parseApiResponse } from "@/lib/utils";
import type { APIResponse } from "@/types/api";

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
  playQueueTime?: number;
  idleTimer: NodeJS.Timeout | null;
  setVoiceActivity: (activity: VoiceActivity) => void;
  connect: (characterId: string) => void;
  start: (payload: ConversationRequest) => void;
  pushUserMessage: (content: string) => void;
  interrupt: () => void;
  clear: () => void;
  resumeAudio: () => void;
}

function int16ToAudioBuffer(
  ctx: AudioContext,
  int16: Int16Array,
  sampleRate = 16000
) {
  const float32 = new Float32Array(int16.length);
  for (let i = 0; i < int16.length; i++) {
    float32[i] = Math.max(-1, Math.min(1, int16[i] / 32768));
  }
  const buffer = ctx.createBuffer(1, float32.length, sampleRate);
  buffer.copyToChannel(float32, 0);
  return buffer;
}

export const useVoiceConversation = create<VoiceConversationState>((set, get) => ({
  connection: "idle",
  ws: null,
  characterId: null,
  history: [],
  audioCtx: null,
  gainNode: null,
  isPlaying: false,
  playQueueTime: undefined,
  idleTimer: null,

  setVoiceActivity: (activity: VoiceActivity) => {
    const { setVoiceActivity } = require("@/store/vad").useVadStore.getState();
    setVoiceActivity(activity);
  },

  connect: (characterId: string) => {
    const { ws, interrupt } = get();
    if (ws && ws.readyState < 2) return; // Already connected or connecting

    interrupt(); // Clean up previous connection if any
    set({ connection: "connecting", characterId });

    const url = new URL(`${process.env.NEXT_PUBLIC_WS_URL}/ws/voice-conversation`);
    const newWs = new WebSocket(url);

    newWs.binaryType = "arraybuffer";

    newWs.onopen = () => {
      set({ connection: "connected", ws: newWs });
    };

    newWs.onmessage = async (ev) => {
      const { idleTimer, setVoiceActivity } = get();
      if (idleTimer) clearTimeout(idleTimer);

      if (typeof ev.data === "string") {
        const result = parseApiResponse(ev.data);
        if (!result.ok) {
          console.error("Invalid message format:", result.reason, ev.data);
          return;
        }

        const parsed = result.value as APIResponse<unknown>;
        if (!parsed.success) {
          const errMsg = parsed.error?.message ?? "API error";
          set((s) => ({ history: [...s.history, { role: "assistant", content: `Error: ${errMsg}` }] }));
          setVoiceActivity(VoiceActivity.Idle); // On error, set to idle
          return;
        }

        const msgs = parsed.data && typeof parsed.data === "object" ? (parsed.data as { messages?: unknown }).messages : undefined;
        if (Array.isArray(msgs)) {
          const additions: ChatMessage[] = msgs
            .map((m): ChatMessage | null => {
              if (m && typeof m === "object" && "role" in m && "content" in m) {
                const roleVal = (m as { role: unknown }).role;
                const contentVal = (m as { content: unknown }).content;
                const role: Role = roleVal === "assistant" ? "assistant" : roleVal === "user" ? "user" : "system";
                return { role, content: String(contentVal ?? "") };
              }
              return null;
            })
            .filter((x): x is ChatMessage => Boolean(x));

          if (additions.length > 0) {
            set((s) => {
              const history = [...s.history];
              for (const add of additions) {
                const last = history[history.length - 1];
                if (add.role === "assistant" && last?.role === "assistant") {
                  history[history.length - 1] = { role: "assistant", content: (last.content ?? "") + add.content };
                } else {
                  history.push(add);
                }
              }
              return { history };
            });
          }
        }
        return;
      }

      const arrayBuf = ev.data as ArrayBuffer;
      const int16 = new Int16Array(arrayBuf);
      const { audioCtx, gainNode } = get();
      if (!audioCtx || !gainNode) return;

      if (!get().isPlaying) {
        setVoiceActivity(VoiceActivity.Speaking);
      }

      try {
        const audioBuffer = int16ToAudioBuffer(audioCtx, int16, 16000);
        const source = audioCtx.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(gainNode);

        const now = audioCtx.currentTime;
        const queuedStart = get().playQueueTime ?? now;
        const startAt = Math.max(queuedStart, now);
        source.start(startAt);
        const nextTime = startAt + audioBuffer.duration;
        set({ isPlaying: true, playQueueTime: nextTime });

        const newIdleTimer = setTimeout(() => {
          set({ isPlaying: false, playQueueTime: undefined, idleTimer: null });
          setVoiceActivity(VoiceActivity.Idle);
        }, (nextTime - now + 0.5) * 1000);
        set({ idleTimer: newIdleTimer });

      } catch (err) {
        console.error("Failed to play pcm chunk", err);
      }
    };

    newWs.onerror = (e) => {
      console.error("voice-conversation ws error", e);
      set({ connection: "disconnected" });
    };

    newWs.onclose = () => {
      set({ connection: "disconnected", ws: null });
    };
  },

  start: (payload: ConversationRequest) => {
    const { ws, connection } = get();
    if (ws && connection === "connected") {
      get().setVoiceActivity(VoiceActivity.Loading);
      ws.send(JSON.stringify(payload));
    } else {
      console.error("WebSocket not connected. Cannot start conversation.");
    }
  },

  pushUserMessage: (content: string) => {
    if (!content) return;
    set((s) => ({ history: [...s.history, { role: "user", content }] }));
  },

  interrupt: () => {
    const { ws, audioCtx, idleTimer } = get();
    if (ws && ws.readyState < 2) {
      try { ws.close(); } catch {}
    }
    if (audioCtx) {
      try { audioCtx.close(); } catch {}
    }
    if (idleTimer) clearTimeout(idleTimer);
    set({ ws: null, audioCtx: null, gainNode: null, isPlaying: false, playQueueTime: undefined, connection: "disconnected", idleTimer: null });
  },

  clear: () => set({ history: [] }),

  resumeAudio: () => {
    let { audioCtx } = get();
    if (!audioCtx) {
      try {
        let Ctx: typeof AudioContext | undefined = window.AudioContext;
        const w = window as unknown as { webkitAudioContext?: typeof AudioContext };
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
}));
