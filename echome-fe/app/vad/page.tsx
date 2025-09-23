"use client";
import { useEffect, useRef, useState } from "react";
import { useVoiceActivity, VoiceActivity } from "../../hooks/useVoiceActivity";
import { AudioAnimation } from "../components/AudioAnimation";

export default function Page() {
    const [audioQueue, setAudioQueue] = useState<Float32Array[]>([]);
    const [isPlaying, setIsPlaying] = useState(false);
    const audioContextRef = useRef<AudioContext | null>(null);
    // 当前正在播放的 AudioBufferSourceNode 引用，用于在用户打断时立即停止播放
    const sourceRef = useRef<AudioBufferSourceNode | null>(null);

    const { activity } = useVoiceActivity({
        onSpeechEnd: (audio: Float32Array) => {
            console.log("Received audio:", audio);
            setAudioQueue(q => [...q, audio]);
        },
        onFrameProcessed: (frame: Float32Array) => {
            console.log("Frame processed:", frame);
        }
    });

    useEffect(() => {
        if (activity === VoiceActivity.Speaking) {
            // 用户开始说话 -> 立即停止当前播放并清空队列
            const current = sourceRef.current;
            if (current) {
                // 移除引用，避免 onended 回调再次处理
                sourceRef.current = null;
                try {
                    current.stop();
                } catch {
                    // ignore
                }
            }
            setAudioQueue([]);
            setIsPlaying(false);
        }
    }, [activity]);

    useEffect(() => {
        if (isPlaying || audioQueue.length === 0) return;

        const playNextAudio = async () => {
            const audioData = audioQueue[0];
            if (!audioContextRef.current) {
                audioContextRef.current = new AudioContext({ sampleRate: 16000 });
            }

            const audioContext = audioContextRef.current;
            const audioBuffer = audioContext.createBuffer(1, audioData.length, 16000);
            audioBuffer.copyToChannel(new Float32Array(audioData), 0);

            const source = audioContext.createBufferSource();
            source.buffer = audioBuffer;
            source.connect(audioContext.destination);

            // 记录当前播放源，供中断时停止
            sourceRef.current = source;
            setIsPlaying(true);

            source.onended = () => {
                // 仅当这是当前活跃的 source 时才处理结束逻辑（避免循环/重复处理）
                if (sourceRef.current === source) {
                    sourceRef.current = null;
                    setAudioQueue(q => q.slice(1));
                    setIsPlaying(false);
                }
            };

            source.start();
        };

        playNextAudio().catch(err => {
            console.warn("播放失败，可能需要用户交互", err);
            setIsPlaying(false);
        });
    }, [audioQueue, isPlaying]);


    return (
        <div className="bg-gray-200 flex items-center justify-center min-h-screen">
            <div className="w-full max-w-3xl h-screen">
                <AudioAnimation activity={activity} />
            </div>
        </div>
    );
}
