"use client";
import { create } from "zustand";
import { useCharacterStore } from "./character";

type Role = "system" | "user" | "assistant";

export type ChatMessage = {
  role: Role;
  content: string;
};

export type ConversationRequest = {
  characterId?: string;
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
  isResponding: boolean;
  sources: AudioBufferSourceNode[];
  nextStartTime?: number;
  idleTimer: NodeJS.Timeout | null;
  reconnectTimer: NodeJS.Timeout | null;
  isInterrupted: boolean;
  echoGuardUntil: number | null;
  onConnectCallbacks: (() => void)[];
  connect: (characterId: string) => void;
  start: (payload: ConversationRequest) => void;
  pushUserMessage: (content: string) => void;
  interrupt: () => void;
  disconnect: () => void;
  stopPlaying: () => void;
  clear: () => void;
  resumeAudio: () => void;
  deleteMessage: (index: number) => void;
  editMessage: (index: number, newContent: string, truncate?: boolean) => void;
  retryLastAssistantMessage: () => void;
}

// 将 Int16 PCM 转为 AudioBuffer
function int16ToAudioBuffer(
  ctx: AudioContext,
  int16: Int16Array,
  sampleRate = 22050,
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
    isResponding: false,
    sources: [],
    nextStartTime: undefined,
    idleTimer: null,
    reconnectTimer: null,
    isInterrupted: false,
    echoGuardUntil: null,
    onConnectCallbacks: [],

    // 建立或重用 WebSocket 连接
    connect: (characterId: string) => {
      const { ws, reconnectTimer } = get();
      if (ws && ws.readyState < 2) return;
      if (reconnectTimer) clearTimeout(reconnectTimer);

      set({ connection: "connecting", characterId, reconnectTimer: null });

      const url = new URL(
        `${process.env.NEXT_PUBLIC_WS_URL}/ws/voice-conversation`,
      );
      // 将 characterId 放到 URL 查询参数中
      if (characterId) url.searchParams.set("characterId", characterId);
      const newWs = new WebSocket(url.toString());

      newWs.binaryType = "arraybuffer";

      newWs.onopen = () => {
        set({ connection: "connected", ws: newWs });

        const { onConnectCallbacks, history, characterId, start } = get();

        if (onConnectCallbacks.length > 0) {
          console.log(`Executing ${onConnectCallbacks.length} queued actions.`);
          for (const cb of onConnectCallbacks) {
            cb();
          }
          set({ onConnectCallbacks: [] });
          return;
        }

        const lastMessage = history[history.length - 1];

        // 如果是重连，并且最后一条消息是用户发的，就自动重试
        if (history.length > 0 && lastMessage?.role === "user" && characterId) {
          console.log(
            "Reconnected. The last message was from the user, automatically retrying.",
          );
          const { currentCharacter } = useCharacterStore.getState();
          const messages = [
            {
              role: "system" as const,
              content:
                currentCharacter?.prompt || "You are a helpful assistant.",
            },
            ...history,
          ];
          start({ characterId, messages });
        }
      };

      newWs.onmessage = async (ev) => {
        if (get().isInterrupted) return;

        set({ isResponding: false });
        const { idleTimer } = get();
        if (idleTimer) clearTimeout(idleTimer);

        // 文本消息处理（流式或完整文本）
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
                const limitedHistory = history.slice(-30);
                return { history: limitedHistory };
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
                const limitedHistory = history.slice(-30);
                return { history: limitedHistory };
              });
            } else if (message.type === "tts_error") {
              console.warn("TTS error ignored:", message.message);
            }
          } catch (e) {
            console.error("Failed to parse ws message", e);
          }
          return;
        }

        // 二进制 PCM 音频处理
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
          const startAt = nextStartTime && nextStartTime > now
            ? nextStartTime
            : now;

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
      };

      newWs.onclose = () => {
        set({ connection: "disconnected", ws: null });
        const { characterId: currentCharacterId } = get();
        if (currentCharacterId) {
          // 断线后尝试重连
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

    // 向服务器发送对话请求（开启流式）
    start: (payload: ConversationRequest) => {
      const { ws, connection, connect, characterId } = get();

      const sendMessage = () => {
        const { ws: currentWs } = get();
        if (currentWs && get().connection === "connected") {
          set({ isInterrupted: false, isResponding: true });
          currentWs.send(JSON.stringify({ ...payload, stream: true }));
        } else {
          console.error("Failed to send message even after connect callback.");
        }
      };

      if (ws && connection === "connected") {
        sendMessage();
      } else {
        console.log(
          "WebSocket not connected. Queuing message and attempting to connect.",
        );
        set((state) => ({
          onConnectCallbacks: [...state.onConnectCallbacks, sendMessage],
        }));

        if (connection !== "connecting") {
          const charId = characterId || payload.characterId;
          if (charId) {
            connect(charId);
          } else {
            console.error("Cannot connect: characterId is missing.");
          }
        }
      }
    },

    pushUserMessage: (content: string) => {
      if (!content) return;
      set((s) => {
        const newHistory = [...s.history, { role: "user" as const, content }];
        const limitedHistory = newHistory.slice(-30);
        return { history: limitedHistory };
      });
    },

    stopPlaying: () => {
      const { sources, idleTimer } = get();
      if (idleTimer) clearTimeout(idleTimer);

      sources.forEach((source) => {
        try {
          source.stop();
        } catch {
          // 忽略已停止的错误
        }
      });

      set({
        isPlaying: false,
        sources: [],
        idleTimer: null,
        nextStartTime: undefined,
      });
    },

    interrupt: () => {
      get().stopPlaying();
      const { ws, reconnectTimer, connect, characterId } = get();

      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }

      if (ws && ws.readyState < 2) {
        ws.onclose = null;
        ws.close();
      }

      // Reset state and immediately start a new connection in the background
      set({
        isInterrupted: true,
        ws: null,
        connection: "idle",
        reconnectTimer: null,
      });

      if (characterId) {
        console.log("Interrupted. Pre-emptively creating a new connection.");
        connect(characterId);
      }
    },

    disconnect: () => {
      const { ws, audioCtx, idleTimer, reconnectTimer } = get();
      if (reconnectTimer) clearTimeout(reconnectTimer);
      set({ characterId: null, reconnectTimer: null });

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

    // 恢复或创建 AudioContext
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

    deleteMessage: (index: number) => {
      set((state) => {
        const history = [...state.history];
        if (
          index < 0 ||
          index >= history.length ||
          history[index].role !== "user"
        ) {
          return state;
        }
        const newHistory = history.slice(0, index);
        return { history: newHistory };
      });
    },

    editMessage: (index: number, newContent: string, truncate = false) => {
      set((state) => {
        const history = [...state.history];
        if (
          index < 0 ||
          index >= history.length ||
          history[index].role !== "user"
        ) {
          return state;
        }
        history[index] = { ...history[index], content: newContent };
        if (truncate) {
          return { history: history.slice(0, index + 1) };
        }
        return { history };
      });
    },

    retryLastAssistantMessage: () => {
      const { history, characterId, start, stopPlaying } = get();
      if (!characterId) return;

      stopPlaying();

      const lastAssistantIndex = history.findLastIndex(
        (msg) => msg.role === "assistant",
      );
      if (lastAssistantIndex === -1) return;

      const newHistory = history.slice(0, lastAssistantIndex);
      set({ history: newHistory });

      const { currentCharacter } = useCharacterStore.getState();
      const messages = [
        {
          role: "system" as const,
          content: currentCharacter?.prompt || "You are a helpful assistant.",
        },
        ...newHistory,
      ];
      start({ characterId, messages });
    },
  }),
);
