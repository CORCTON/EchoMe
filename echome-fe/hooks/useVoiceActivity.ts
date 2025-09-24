import { useMicVAD } from "@ricky0123/vad-react";
import { useEffect } from "react";
import { useVadStore } from "@/store/vad";
import { VoiceActivity } from "@/types/vad";

export interface UseVoiceActivityOptions {
  onSpeechEnd?: (audio: Float32Array) => void;
  onFrameProcessed?: (frame: Float32Array) => void;
}

export function useVoiceActivity(options: UseVoiceActivityOptions) {
  const { setVoiceActivity } = useVadStore();

  const vad = useMicVAD({
    model: "v5",
    baseAssetPath: "/vad/",
    onnxWASMBasePath: "/vad/",
    preSpeechPadMs: 200,
    positiveSpeechThreshold: 0.8,
    minSpeechMs: 100,
    onSpeechStart: () => {
      setVoiceActivity(VoiceActivity.Speaking);
    },
    onSpeechEnd: (audio) => {
      setVoiceActivity(VoiceActivity.Idle);
      options.onSpeechEnd?.(audio);
    },
    onFrameProcessed: (probabilities, frame) => {
      if (probabilities.isSpeech > 0.8) {
        options.onFrameProcessed?.(frame);
      }
    },
  });

  useEffect(() => {
    setVoiceActivity(vad.loading ? VoiceActivity.Loading : VoiceActivity.Idle);
  }, [vad.loading, setVoiceActivity]);

  return { vad };
}
