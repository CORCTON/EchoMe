import { create } from "zustand";
import { VoiceActivity } from "@/types/vad";
import { MicVAD } from "@ricky0123/vad-web";

// 使用 zustand 管理 VAD（语音活动检测）和 ASR（语音识别）相关状态
// 包括：VAD 实例管理、音频预缓冲、ASR websocket 连接与消息处理

export enum ConnectionState {
  Connecting,
  Connected,
  Disconnected,
}

// 从 ASR 服务返回的结果格式
interface AsrResult {
  type: "asr_result";
  text: string;
  sentence_end: boolean;
}

// VAD store 的状态接口（主要字段已命名为可读的含义）
interface VadState {
  isVadReady: boolean; // VAD 是否初始化完成
  asrConnectionState: ConnectionState; // ASR websocket 状态
  voiceActivity: VoiceActivity; // 当前语音活动状态（Idle/Loading/Speaking）
  committedTranscript: string; // 已确认（最终）的识别文本
  transcript: string; // 当前正在拼接的识别文本
  isFinal: boolean; // 当前片段是否为最终结果
  isTranscribing: boolean; // 是否正在把帧发送给 ASR
  socket: WebSocket | null; // ASR websocket
  vad: MicVAD | null; // VAD 实例
  preSpeechBuffer: Float32Array[]; // 语音开始前的预缓冲帧（用于补发）
  audioQueue: Float32Array[]; // 在 websocket 建立期间排队的音频帧
  initVad: (onSpeechEnd: (transcript: string) => void) => Promise<void>; // 初始化 VAD
  disconnect: () => void; // 断开并清理资源
  send: (data: Float32Array) => void; // 发送 PCM 音频到 ASR
  setVoiceActivity: (activity: VoiceActivity) => void; // 更新语音活动状态
  resetTranscript: () => void; // 重置转录文本
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

  // 初始化 VAD：创建 MicVAD，设置回调（开始/结束/帧处理）并启动
  initVad: async (onSpeechEndCallback) => {
    set({ voiceActivity: VoiceActivity.Loading, isVadReady: false });
    try {
      const vad = await MicVAD.new({
        baseAssetPath: "/vad/",
        onnxWASMBasePath: "/vad/",
        model: "v5",
        preSpeechPadMs: 200, // 预缓冲时间
        positiveSpeechThreshold: 0.9, // 语音检测阈值
        negativeSpeechThreshold: 0.5, // 静音检测阈值
        minSpeechMs: 300, // 最小语音长度
        redemptionMs: 750, // 等待静音时间

        // 语音开始：检查 echo guard，打断当前播放，合并并发送预缓冲数据
        onSpeechStart: () => {
          const { echoGuardUntil, interrupt } =
            require("@/store/voice-conversation").useVoiceConversation.getState();
          if (echoGuardUntil && Date.now() < echoGuardUntil) {
            return;
          }
          // 中断当前播放并重置转录状态
          interrupt();
          get().resetTranscript();
          const { preSpeechBuffer, send } = get();
          if (preSpeechBuffer.length > 0) {
            // 将预缓冲帧拼接成一个连续的 Float32Array 并发送
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

        // 语音结束：关闭 ASR 连接并回调最终文本
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

        // 每帧处理：如果正在转录则发送，否则将帧保存在预缓冲队列中
        onFrameProcessed: (_, frame) => {
          const { isTranscribing, preSpeechBuffer, send } = get();
          if (isTranscribing) {
            send(frame);
          } else {
            const bufferSize = 7; // 预缓冲最大帧数
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

  // 断开并清理 websocket 与 VAD 实例
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

  // 发送音频数据到 ASR：先把 Float32 转为 Int16，再通过 websocket 发送
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

    // 如果没有可用 socket 或者正在关闭，则创建新的 websocket 并在 onopen 时发送当前帧与队列中的数据
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
        sendData(newSocket, data); // 发送初始化数据（预缓冲）
        const { audioQueue } = get();
        for (const queuedData of audioQueue) {
          sendData(newSocket, queuedData);
        }
        set({ audioQueue: [] });
      };

      // 处理来自 ASR 的文本结果
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
      // socket 建立中：排队等待发送
      set((state) => ({ audioQueue: [...state.audioQueue, data] }));
    } else if (socket.readyState === WebSocket.OPEN) {
      // 直接发送
      sendData(socket, data);
    }
  },

  setVoiceActivity: (activity: VoiceActivity) => {
    set({ voiceActivity: activity });
  },

  resetTranscript: () =>
    set({ transcript: "", committedTranscript: "", isFinal: false }),
}));
