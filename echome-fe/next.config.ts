import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
const CopyPlugin = require("copy-webpack-plugin");

const nextConfig: NextConfig = {
  output: "standalone",
  webpack: (config) => {
    config.plugins.push(
      new CopyPlugin({
        patterns: [
          {
            from: "node_modules/onnxruntime-web/dist/*.wasm",
            to: "../public/vad/[name][ext]",
          },
          {
            from:
              "node_modules/@ricky0123/vad-web/dist/vad.worklet.bundle.min.js",
            to: "../public/vad/[name][ext]",
          },
          {
            from: "node_modules/@ricky0123/vad-web/dist/*.onnx",
            to: "../public/vad/[name][ext]",
          },
          {
            from: "node_modules/onnxruntime-web/dist/*.mjs",
            to: "../public/vad/[name][ext]",
          },
        ],
      }),
    );

    return config;
  },
};

const withNextIntl = createNextIntlPlugin("./configs/i18n.ts");
export default withNextIntl(nextConfig);
