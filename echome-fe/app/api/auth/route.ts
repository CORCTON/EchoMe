import { type NextRequest, NextResponse } from "next/server";
import { SignJWT } from "jose";

const secret = new TextEncoder().encode(
  process.env.AUTH_SECRET || "fallback-secret-key",
);

// 记录登录尝试
interface AttemptRecord {
  count: number;
  timestamp: number;
}
const loginAttempts = new Map<string, AttemptRecord>();
const MAX_ATTEMPTS = 3;
const ONE_DAY_IN_MS = 24 * 60 * 60 * 1000;

export async function POST(request: NextRequest) {
  const ip = (request.headers.get("x-forwarded-for") ?? "127.0.0.1").split(
    ",",
  )[0];

  let record = loginAttempts.get(ip);
  const now = Date.now();

  // 清理过期记录
  if (record && now - record.timestamp > ONE_DAY_IN_MS) {
    loginAttempts.delete(ip);
    record = undefined;
  }

  if (record && record.count >= MAX_ATTEMPTS) {
    return NextResponse.json(
      { error: "尝试次数过多，请24小时后再试" },
      { status: 429 },
    );
  }

  try {
    const { password } = await request.json();

    // 验证密码
    if (password !== process.env.AUTH_PASSWORD) {
      const newCount = (record?.count || 0) + 1;
      loginAttempts.set(ip, {
        count: newCount,
        timestamp: record?.timestamp || now,
      });
      return NextResponse.json({ error: "密码错误" }, { status: 401 });
    }

    // 登录成功，清除尝试记录
    loginAttempts.delete(ip);

    // 创建JWT token
    const token = await new SignJWT({ authenticated: true })
      .setProtectedHeader({ alg: "HS256" })
      .setIssuedAt()
      .setExpirationTime("24h")
      .sign(secret);

    // 创建响应并设置cookie
    const response = NextResponse.json({ success: true });
    response.cookies.set("auth-token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24, // 24小时
      path: "/",
    });

    return response;
  } catch {
    return NextResponse.json({ error: "服务器错误" }, { status: 500 });
  }
}

export async function DELETE() {
  // 登出逻辑
  const response = NextResponse.json({ success: true });
  response.cookies.delete("auth-token");
  return response;
}
