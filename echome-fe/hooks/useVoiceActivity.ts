import { useMicVAD } from "@ricky0123/vad-react";
import { useEffect, useRef, useState } from "react";

// 预挑可用的 mimeType（按浏览器常见支持顺序）
const PICK_MIME_TYPES = ["audio/webm;codecs=opus", "audio/ogg;codecs=opus", "audio/webm", "audio/ogg"];

function pickSupportedMimeType(): string | undefined {
  return PICK_MIME_TYPES.find(MediaRecorder.isTypeSupported);
}

export enum VoiceActivity {
  Loading,
  Speaking,
  Idle,
}

export interface UseVoiceActivityOptions {
  onChunk?: (chunk: Blob) => void;
}

export function useVoiceActivity(options: UseVoiceActivityOptions) {
  const [activity, setActivity] = useState(VoiceActivity.Loading);
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
        options.onChunk?.(ev.data);
      }
    };
    r.onstop = () => { recorderRef.current = null; setActivity(VoiceActivity.Idle); };
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
      setActivity(VoiceActivity.Speaking);
      startSegmentRecorder();
    },
    onSpeechEnd() {
      stopSegmentRecorder();
    },
  });

  useEffect(() => {
    setActivity(vad.loading ? VoiceActivity.Loading : VoiceActivity.Idle);
  }, [vad.loading]);

  return { activity, vad };
}
