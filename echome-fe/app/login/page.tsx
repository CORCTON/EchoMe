"use client";

import { useTranslations } from "next-intl";
import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { SubmitInput } from "@/components/submit-input";

type LoginStatus = "idle" | "loading" | "error" | "success";

export default function LoginPage() {
  const t = useTranslations("login");
  const [password, setPassword] = useState("");
  const [status, setStatus] = useState<LoginStatus>("idle");
  const [message, setMessage] = useState("");
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirectTo = searchParams.get("redirect") || "/";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatus("loading");
    setMessage("");

    try {
      const response = await fetch("/api/auth", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ password }),
      });

      const data = await response.json();

      if (response.ok) {
        setStatus("success");
        setMessage(t("loginSuccess"));
        // 登录成功，短暂延迟后重定向
        setTimeout(() => {
          router.push(redirectTo);
          router.refresh();
        }, 1000);
      } else {
        setStatus("error");
        setMessage(data.error || t("loginFailed"));
      }
    } catch {
      setStatus("error");
      setMessage(t("networkError"));
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setPassword(e.target.value);
    // 重置状态当用户开始输入新内容
    if (status === "error") {
      setStatus("idle");
      setMessage("");
    }
  };

  const getInputStatus = () => {
    if (status === "loading") return "loading";
    if (status === "error") return "error";
    if (status === "success") return "success";
    return "default";
  };

  const getDescriptionText = () => {
    if (message) return message;
    return t("description");
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            {t("title")}
          </h2>
          <p className="mt-2 text-center text-sm transition-colors duration-200 font-medium">
            <span
              className={
                status === "error"
                  ? "text-red-600"
                  : status === "success"
                    ? "text-green-600"
                    : "text-gray-600"
              }
            >
              {getDescriptionText()}
            </span>
          </p>
        </div>

        <div className="mt-8">
          <SubmitInput
            placeholders={[t("passwordPlaceholder")]}
            onChange={handleChange}
            onSubmit={handleSubmit}
            status={getInputStatus()}
            shouldAnimate={status === "success"}
          />
        </div>
      </div>
    </div>
  );
}
