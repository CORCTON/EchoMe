import { type NextRequest, NextResponse } from "next/server";
import { jwtVerify } from "jose";

const secret = new TextEncoder().encode(process.env.AUTH_SECRET);

// 不需要身份验证的路径
const publicPaths: string[] = ["/login"];

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // 检查是否是公开路径
  if (publicPaths.some((path) => pathname.startsWith(path))) {
    return NextResponse.next();
  }

  // 检查静态资源
  if (
    pathname.startsWith("/_next") ||
    pathname.startsWith("/favicon.ico") ||
    pathname.startsWith("/vad") ||
    pathname.includes(".")
  ) {
    return NextResponse.next();
  }

  // 获取认证token
  const token = request.cookies.get("auth-token")?.value;

  if (!token) {
    // 重定向到登录页面
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  try {
    // 验证JWT token
    await jwtVerify(token, secret);
    return NextResponse.next();
  } catch {
    // Token无效，重定向到登录页面
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    const response = NextResponse.redirect(loginUrl);
    response.cookies.delete("auth-token");
    return response;
  }
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
