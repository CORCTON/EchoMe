import { useMicVAD } from "@ricky0123/vad-react";
import { useEffect, useState } from "react";

export enum VoiceActivity {
  Loading,
  Speaking,
  Idle,
}

export interface UseVoiceActivityOptions {
  onSpeechEnd?: (audio: Float32Array) => void;
  onFrameProcessed?: (frame: Float32Array) => void;
}

export function useVoiceActivity(options: UseVoiceActivityOptions) {
  const [activity, setActivity] = useState(VoiceActivity.Loading);

  const vad = useMicVAD({
    model: "v5",
    baseAssetPath: "/vad/",
    onnxWASMBasePath: "/vad/",
    preSpeechPadMs: 200,
    positiveSpeechThreshold: 0.8,
    minSpeechMs: 100,
    onSpeechStart: () => {
      setActivity(VoiceActivity.Speaking);
    },
    onSpeechEnd: (audio) => {
      setActivity(VoiceActivity.Idle);
      options.onSpeechEnd?.(audio);
    },
    onFrameProcessed: (probabilities, frame) => {
      if (probabilities.isSpeech > 0.8) {
        options.onFrameProcessed?.(frame);
      }
    },
  });

  useEffect(() => {
    setActivity(vad.loading ? VoiceActivity.Loading : VoiceActivity.Idle);
  }, [vad.loading]);

  return { activity, vad };
}
