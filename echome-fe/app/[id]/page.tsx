"use client";
import {
    Alignment,
    Fit,
    Layout,
    StateMachineInputType,
    useRive,
} from "@rive-app/react-canvas-lite";
import { useEffect, useState } from "react";
import { useMicVAD } from "@ricky0123/vad-react";
import OpusMediaRecorder from "opus-media-recorder";


const options = { mimeType: "audio/ogg; codecs=opus" };
const workerOptions = {
    encoderWorkerFactory: () =>
        new Worker(
            new URL(
                "opus-media-recorder/encoderWorker.umd.js",
                import.meta.url,
            ),
        ),
    OggOpusEncoderWasmPath: new URL(
        "opus-media-recorder/OggOpusEncoder.wasm",
        import.meta.url,
    ),
    WebMOpusEncoderWasmPath: new URL(
        "opus-media-recorder/WebMOpusEncoder.wasm",
        import.meta.url,
    ),
};

export default function Page() {
    const vad = useMicVAD({
        model: "v5",
        baseAssetPath: "/vad/",
        onnxWASMBasePath: "/vad/",
        onSpeechRealStart: () => {
            setMachineNumber(1);
        },
        onSpeechEnd(audio) {
            const audioContext = new AudioContext();
            const audioBuffer = audioContext.createBuffer(
                1,
                audio.length,
                audioContext.sampleRate,
            );
            audioBuffer.copyToChannel(new Float32Array(audio), 0);

            const source = audioContext.createBufferSource();
            source.buffer = audioBuffer;

            const dest = audioContext.createMediaStreamDestination();
            source.connect(dest);

            const mediaRecorder = new OpusMediaRecorder(
                dest.stream,
                options,
                workerOptions,
            );

            mediaRecorder.addEventListener(
                "dataavailable",
                (event: BlobEvent) => {
                    const reader = new FileReader();
                    reader.onload = () => {
                        console.log("Opus binary data:", reader.result);
                    };
                    reader.readAsArrayBuffer(event.data);
                },
            );

            mediaRecorder.start();
            source.start();

            source.onended = () => {
                mediaRecorder.stop();
                audioContext.close();
                setMachineNumber(2);
            };
        },
    });

    const { RiveComponent, rive } = useRive({
        src: "/ai_voice_states.riv",
        animations: ["listen", "speak", "think"],
        stateMachines: "StateMachine",
        layout: new Layout({
            fit: Fit.Contain,
            alignment: Alignment.Center,
        }),
        autoplay: true,
    });

    const [machineNumber, setMachineNumber] = useState<number>(0);

    useEffect(() => {
        if (vad.loading) {
            setMachineNumber(0);
        } else {
            setMachineNumber(2);
        }
    }, [vad.loading]);

    useEffect(() => {
        if (!rive) return;

        const inputs = rive.stateMachineInputs("StateMachine");
        if (!inputs) return;

        const numberInputs = inputs.filter((i) =>
            i.type === StateMachineInputType.Number
        );
        if (numberInputs.length === 0) return;

        numberInputs.forEach((i) => {
            i.value = machineNumber;
        });
    }, [machineNumber, rive]);

    return (
        <div className="bg-gray-200 flex items-center justify-center min-h-screen">
            <div className="w-full max-w-3xl h-screen">
                <RiveComponent />
            </div>
        </div>
    );
}
