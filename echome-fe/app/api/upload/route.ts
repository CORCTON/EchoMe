import { type NextRequest, NextResponse } from "next/server";
import OSS from "ali-oss";
import { v4 as uuidv4 } from "uuid";

// 阿里 OSS 客户端
const client = new OSS({
  region: process.env.OSS_REGION,
  accessKeyId: process.env.OSS_ACCESS_KEY_ID,
  accessKeySecret: process.env.OSS_ACCESS_KEY_SECRET,
  bucket: process.env.OSS_BUCKET,
  secure: true,
});

// 简单内存限流（按 IP）
type RateEntry = { cnt: number; start: number };
const WINDOW_MS = 60_000;
const MAX_PER_MIN = parseInt(process.env.MAX_UPLOADS_PER_MINUTE || "10", 10);
const rateMap = new Map<string, RateEntry>();

function isRateLimited(ip: string) {
  const now = Date.now();
  const e = rateMap.get(ip);
  if (!e) {
    rateMap.set(ip, { cnt: 1, start: now });
    return false;
  }

  if (now - e.start > WINDOW_MS) {
    rateMap.set(ip, { cnt: 1, start: now });
    return false;
  }

  e.cnt += 1;
  rateMap.set(ip, e);
  return e.cnt > MAX_PER_MIN;
}

// 定期清理过期记录
setInterval(() => {
  const now = Date.now();
  for (const [ip, e] of rateMap.entries()) {
    if (now - e.start > WINDOW_MS * 5) rateMap.delete(ip);
  }
}, WINDOW_MS * 2);

const MAX_SIZE = parseInt(
  process.env.MAX_UPLOAD_SIZE_BYTES || `${10 * 1024 * 1024}`,
  10,
);
const ALLOWED = (
  process.env.ALLOWED_UPLOAD_TYPES || "audio/mpeg,audio/wav,application/pdf"
).split(",");

function allowedType(mime: string) {
  if (!mime) return false;
  if (mime.startsWith("image/")) return true;
  return ALLOWED.includes(mime);
}

export async function POST(req: NextRequest) {
  try {
    const ip =
      req.headers.get("x-forwarded-for") ||
      req.headers.get("x-real-ip") ||
      "unknown";
    if (isRateLimited(ip))
      return NextResponse.json({ error: "Too many requests" }, { status: 429 });

    const form = await req.formData();
    const file = form.get("file") as File | null;
    if (!file)
      return NextResponse.json({ error: "No file provided" }, { status: 400 });
    if (!allowedType(file.type))
      return NextResponse.json(
        { error: "File type not allowed" },
        { status: 400 },
      );
    if (file.size > MAX_SIZE)
      return NextResponse.json({ error: "File too large" }, { status: 400 });

    const buffer = Buffer.from(await file.arrayBuffer());
    const name = `${uuidv4()}-${file.name}`;
    const res = await client.put(name, buffer);
    return NextResponse.json({ url: res.url });
  } catch (err) {
    console.error("上传错误：", err);
    return NextResponse.json({ error: "Upload failed" }, { status: 500 });
  }
}
