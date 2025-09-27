import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
const CopyPlugin = require("copy-webpack-plugin");

// 阿里云 oss 配置
const ossUrl =
  `${process.env.OSS_BUCKET}.${process.env.OSS_REGION}.aliyuncs.com`;

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: ossUrl,
      },
      {
        protocol: "https",
        hostname: "api.dicebear.com",
      },
    ],
  },
  async rewrites() {
    return [
      {
        source: "/v1/api/:path*",
        destination: `${process.env.API_BASE_URL}/api/:path*`,
      },
    ];
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
