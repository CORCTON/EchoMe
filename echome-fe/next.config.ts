import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
const CopyPlugin = require("copy-webpack-plugin");

const nextConfig: NextConfig = {
  output: "standalone",
  images:{
    remotePatterns:[
      {
        protocol: 'https',
        hostname: 'images.unsplash.com',
      }
    ]
  },
  webpack: (config) => {
    config.ignoreWarnings = [
        ...(config.ignoreWarnings || []),
        /Critical dependency: require function is used in a way in which dependencies cannot be statically extracted/,
    ];
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
