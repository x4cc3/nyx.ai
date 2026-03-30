"use client";

import { useEffect, useRef, useState } from "react";

interface PollingOptions<T> {
  pauseWhenHidden?: boolean;
  stopWhen?: (data: T) => boolean;
}

export function usePolling<T>(
  fn: () => Promise<T>,
  intervalMs: number,
  deps: ReadonlyArray<unknown> = [],
  enabled = true,
  options: PollingOptions<T> = {}
) {
  const { pauseWhenHidden = true, stopWhen } = options;
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [stopped, setStopped] = useState(false);
  const [visible, setVisible] = useState(true);
  const fnRef = useRef(fn);
  const stopWhenRef = useRef(stopWhen);

  useEffect(() => {
    fnRef.current = fn;
    stopWhenRef.current = stopWhen;
  });

  useEffect(() => {
    setStopped(false);
  }, [deps]);

  useEffect(() => {
    if (!pauseWhenHidden || typeof document === "undefined") {
      return;
    }

    const handleVisibility = () => {
      setVisible(document.visibilityState === "visible");
    };

    handleVisibility();
    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [pauseWhenHidden]);

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      return;
    }

    if (stopped || (pauseWhenHidden && !visible)) {
      setLoading(false);
      return;
    }

    let active = true;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const tick = async () => {
      let shouldStop = false;

      try {
        const result = await fnRef.current();
        if (active) {
          setData(result);
          setError(null);
          const stopCheck = stopWhenRef.current;
          shouldStop = stopCheck ? stopCheck(result) : false;
        }
      } catch (err) {
        if (active) {
          setError(err as Error);
        }
      } finally {
        if (active) {
          setLoading(false);
          if (shouldStop) {
            setStopped(true);
            return;
          }

          timeoutId = setTimeout(tick, intervalMs);
        }
      }
    };

    tick();
    return () => {
      active = false;
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [enabled, intervalMs, pauseWhenHidden, visible, stopped, deps]);

  return { data, error, loading };
}
