import * as React from "react";
import { cn } from "@/lib/cn";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        ref={ref}
        type={type}
        className={cn(
          [
            "flex h-11 w-full rounded-lg border border-input bg-background/20 px-3 py-2 text-sm text-foreground",
            "placeholder:text-muted-foreground/70",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            "disabled:cursor-not-allowed disabled:opacity-50",
          ].join(" "),
          className
        )}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";

export { Input };

