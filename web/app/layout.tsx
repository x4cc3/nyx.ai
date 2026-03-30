import "./globals.css";
import type { Metadata } from "next";
import localFont from "next/font/local";
import AppProviders from "@/components/i18n/AppProviders";
import { getMetadataBase } from "@/lib/env";

const spaceGrotesk = localFont({
  src: [
    {
      path: "../public/fonts/SpaceGrotesk.ttf",
      style: "normal",
      weight: "300 700",
    },
  ],
  variable: "--font-sans",
  display: "swap",
});

const jetbrainsMono = localFont({
  src: [
    {
      path: "../public/fonts/JetBrainsMono.ttf",
      style: "normal",
      weight: "300 800",
    },
  ],
  variable: "--font-mono",
  display: "swap",
});

export const metadata: Metadata = {
  metadataBase: getMetadataBase(),
  title: "NYX",
  description: "Autonomous AI pentest control plane"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      className={`${spaceGrotesk.variable} ${jetbrainsMono.variable}`}
      suppressHydrationWarning
    >
      <body className={spaceGrotesk.className} suppressHydrationWarning>
        <div className="landing-bg-shell" aria-hidden="true">
          <div className="landing-bg-base absolute inset-0" />
          <div className="landing-grid-overlay" />
          <div className="landing-frame-overlay" />
        </div>
        <div className="relative z-[1]">
          <AppProviders>{children}</AppProviders>
        </div>
      </body>
    </html>
  );
}
