"use client";

import { useEffect, type RefObject } from "react";

export function useOnClickOutside<T extends HTMLElement>(
  ref: RefObject<T | null>,
  onOutside: () => void
) {
  useEffect(() => {
    const onMouseDown = (event: MouseEvent) => {
      const element = ref.current;
      if (!element) {
        return;
      }
      if (event.target instanceof Node && element.contains(event.target)) {
        return;
      }
      onOutside();
    };
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [onOutside, ref]);
}
