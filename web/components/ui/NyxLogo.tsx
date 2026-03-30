import React from "react";

export function NyxLogo(props: React.SVGProps<SVGSVGElement>) {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 512 512"
            width="1em"
            height="1em"
            {...props}
        >
            <defs>
                <linearGradient id="nyx-bg" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stopColor="#0E0A1F" />
                    <stop offset="100%" stopColor="#020108" />
                </linearGradient>
                <linearGradient id="nyx-n" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stopColor="#4ade80" />
                    <stop offset="50%" stopColor="#22c55e" />
                    <stop offset="100%" stopColor="#166534" />
                </linearGradient>
                <filter id="nyx-glow" x="-20%" y="-20%" width="140%" height="140%">
                    <feGaussianBlur stdDeviation="15" result="blur" />
                    <feComposite in="SourceGraphic" in2="blur" operator="over" />
                </filter>
            </defs>

            <rect width="512" height="512" rx="112" fill="url(#nyx-bg)" />

            <polygon
                points="256 64 422 160 422 352 256 448 90 352 90 160"
                fill="none"
                stroke="rgba(255,255,255,0.15)"
                strokeWidth="12"
            />
            <polygon
                points="256 96 394 176 394 336 256 416 118 336 118 176"
                fill="none"
                stroke="rgba(255,255,255,0.25)"
                strokeWidth="6"
            />

            <g filter="url(#nyx-glow)">
                <path
                    d="M160 368 L160 144"
                    fill="none"
                    stroke="url(#nyx-n)"
                    strokeWidth="48"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
                <path
                    d="M160 144 L352 368"
                    fill="none"
                    stroke="url(#nyx-n)"
                    strokeWidth="48"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
                <path
                    d="M352 368 L352 144"
                    fill="none"
                    stroke="url(#nyx-n)"
                    strokeWidth="48"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
            </g>
        </svg>
    );
}
