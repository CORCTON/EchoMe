"use client";
import { create } from "zustand";
import { useCharacterStore } from "./character";
import type { FileObject } from "./file";

type Role = "system" | "user" | "assistant";

export type ImageContent = {
  type: "image_url";
  image_url: { url: string };
};

export type TextContent = {
  type: "text";
  text: string;
};

export type MessageContent = string | (ImageContent | TextContent)[];

export type ChatMessage = {
  role: Role;
  content: MessageContent;
};

export type ConversationRequest = {
  messages: ChatMessage[];
  enable_search?: boolean;
};

type ConnectionState = "idle" | "connecting" | "connected" | "disconnected";

export interface VoiceConversationState {
  connection: ConnectionState;
  ws: WebSocket | null;
  characterId: string | null;
  history: ChatMessage[];
  files: FileObject[];
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
  setFiles: (files: FileObject[]) => void;
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
    files: [],
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
      const { ws, reconnectTimer, characterId: currentCharacterId } = get();

      // 如果存在一个活动的 ws 连接
      if (ws && ws.readyState < 2) {
        // 如果 characterId 相同，则无需操作
        if (characterId === currentCharacterId) {
          return;
        }
        ws.onclose = null; // 防止触发重连逻辑
        ws.close();
      }

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

        const { onConnectCallbacks, history, characterId, start, files } = get();

        if (onConnectCallbacks.length > 0) {
          for (const cb of onConnectCallbacks) {
            cb();
          }
          set({ onConnectCallbacks: [] });
          return;
        }

        const lastMessage = history[history.length - 1];

        // 如果是重连，并且最后一条消息是用户发的，就自动重试
        if (history.length > 0 && lastMessage?.role === "user" && characterId) {
          const { currentCharacter } = useCharacterStore.getState();
          
          const messages = [...history];

          const systemPrompt = {
            role: "system" as const,
            content: currentCharacter?.prompt || "You are a helpful assistant.",
          };

          start({ messages: [systemPrompt, ...messages] });
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
                if (last?.role === "assistant" && typeof last.content === 'string') {
                  history[history.length - 1] = {
                    role: "assistant",
                    content: last.content + message.content,
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
                if (last?.role === "assistant" && typeof last.content === 'string') {
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
          console.warn(
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
      const { ws, connection, connect, characterId, files } = get();

      const sendMessage = () => {
        const { ws: currentWs } = get();
        if (currentWs && get().connection === "connected") {
          set({ isInterrupted: false, isResponding: true });

          const { modelSettings } = useCharacterStore.getState();
          const { messages } = payload;
          
          const finalPayload: { 
            messages: ChatMessage[];
            stream: boolean;
            enable_search?: boolean;
          } = { 
            messages,
            stream: true,
          };

          if (modelSettings.internetAccess) {
            finalPayload.enable_search = true;
          }

          const userMessages = finalPayload.messages.filter(msg => msg.role === 'user');
          const isFirstUserMessage = userMessages.length === 1;
          
          if (isFirstUserMessage && files.length > 0) {
            const firstUserMessageIndex = finalPayload.messages.findIndex(msg => msg.role === 'user');
            if (firstUserMessageIndex !== -1) {
              const firstUserMessage = finalPayload.messages[firstUserMessageIndex];
              if (typeof firstUserMessage.content === 'string') {
                const imageContent: ImageContent[] = files.map(file => ({
                  type: 'image_url',
                  image_url: { url: file.url },
                }));
                const textContent: TextContent = { type: 'text', text: firstUserMessage.content };
                const newContent = [...imageContent, textContent];
                
                finalPayload.messages[firstUserMessageIndex] = {
                  ...firstUserMessage,
                  content: newContent,
                };

                // Also update the history in the store
                set(state => {
                  const newHistory = [...state.history];
                  const messageToUpdate = newHistory.find(msg => msg.role === 'user' && msg.content === firstUserMessage.content);
                  if (messageToUpdate) {
                    messageToUpdate.content = newContent;
                  }
                  return { history: newHistory };
                });
              }
            }
          }
          
          currentWs.send(JSON.stringify(finalPayload));
        } else {
          console.error("Failed to send message even after connect callback.");
        }
      };

      if (ws && connection === "connected") {
        sendMessage();
      } else {
        console.warn(
          "WebSocket not connected. Queuing message and attempting to connect.",
        );
        set((state) => ({
          onConnectCallbacks: [...state.onConnectCallbacks, sendMessage],
        }));

        if (connection !== "connecting") {
          if (characterId) {
            connect(characterId);
          } else {
            console.error("Cannot connect: characterId is missing from the store.");
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

      set({
        isInterrupted: true,
        ws: null,
        connection: "idle",
        reconnectTimer: null,
      });

      if (characterId) {
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
        files: [],
      });
    },

    clear: () => set({ history: [], files: [] }),

    setFiles: (files: FileObject[]) => set({ files }),

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
        const currentMessage = history[index];
        if (Array.isArray(currentMessage.content)) {
          const textPart = currentMessage.content.find(
            (part) => part.type === "text",
          ) as TextContent | undefined;
          if (textPart) {
            textPart.text = newContent;
          }
        } else {
          history[index] = { ...currentMessage, content: newContent };
        }

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
      start({ messages });
    },
  }),
);
