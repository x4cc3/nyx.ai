"use client";

import * as React from "react";
import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { cn } from "@/lib/cn";
import { Input } from "@/components/ui";
import { useLanguage } from "@/components/i18n/LanguageProvider";

type PasswordFieldProps = Omit<React.ComponentProps<typeof Input>, "type">;

export default function PasswordField({ className, disabled, ...props }: PasswordFieldProps) {
  const { t } = useLanguage();
  const [visible, setVisible] = useState(false);

  return (
    <div className="relative">
      <Input
        {...props}
        disabled={disabled}
        type={visible ? "text" : "password"}
        className={cn("pr-10", className)}
      />
      <button
        type="button"
        disabled={disabled}
        aria-label={visible ? t("auth.password.hide") : t("auth.password.show")}
        aria-pressed={visible}
        onClick={() => setVisible((v) => !v)}
        className={cn(
          [
            "absolute right-2 top-1/2 -translate-y-1/2 rounded-md p-1",
            "text-muted-foreground hover:text-foreground",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            "disabled:opacity-50",
          ].join(" ")
        )}
      >
        {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  );
}
