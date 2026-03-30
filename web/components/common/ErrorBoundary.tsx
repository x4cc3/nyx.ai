"use client";

import React from "react";
import { AlertCircle, RefreshCw } from "lucide-react";

interface ErrorBoundaryProps {
  children: React.ReactNode;
  /** Optional fallback to render instead of the default error card. */
  fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Catches render errors in child components and displays a
 * recoverable error card instead of a white screen.
 *
 * Usage:
 * ```tsx
 * <ErrorBoundary>
 *   <MyComponent />
 * </ErrorBoundary>
 * ```
 */
export default class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  private fallbackRef = React.createRef<HTMLDivElement>();

  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    if (process.env.NODE_ENV === "development") {
      console.error(
        "[ErrorBoundary] Uncaught error:",
        error,
        info.componentStack,
      );
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null });
  };

  componentDidUpdate(_: ErrorBoundaryProps, prevState: ErrorBoundaryState) {
    if (!prevState.hasError && this.state.hasError) {
      this.fallbackRef.current?.focus();
    }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;

      return (
        <div className="flex items-center justify-center min-h-[300px] p-6">
          <div
            ref={this.fallbackRef}
            role="alert"
            tabIndex={-1}
            className="max-w-md w-full rounded-xl border border-destructive/30 bg-card/80 backdrop-blur-lg p-6 space-y-4 shadow-lg"
          >
            <div className="flex items-center gap-3 text-destructive">
              <AlertCircle className="h-6 w-6 shrink-0" />
              <h2 className="text-lg font-semibold">Something went wrong</h2>
            </div>

            <p className="text-sm text-muted-foreground leading-relaxed">
              An unexpected error occurred while rendering this section. You can
              try reloading the component or refreshing the page.
            </p>

            {this.state.error && (
              <pre className="text-xs font-mono text-destructive/80 bg-muted rounded-lg p-3 overflow-x-auto max-h-32">
                {this.state.error.message}
              </pre>
            )}

            <button
              type="button"
              onClick={this.handleRetry}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <RefreshCw className="h-4 w-4" />
              Try Again
            </button>
            {this.state.error && (
              <button
                type="button"
                onClick={() => {
                  const msg = this.state.error?.message || "Unknown error";
                  navigator.clipboard?.writeText(msg).catch(() => {
                    /* clipboard not available */
                  });
                }}
                className="inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted/20 transition-colors"
              >
                Copy Error
              </button>
            )}
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
