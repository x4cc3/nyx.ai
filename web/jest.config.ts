import type { Config } from "jest";

const config: Config = {
    testEnvironment: "jsdom",
    transform: {
        "^.+\\.tsx?$": [
            "ts-jest",
            {
                tsconfig: "tsconfig.json",
                jsx: "react-jsx",
            },
        ],
    },
    moduleNameMapper: {
        "^@/(.*)$": "<rootDir>/$1",
    },
    modulePathIgnorePatterns: ["<rootDir>/.next/"],
    setupFilesAfterEnv: ["<rootDir>/jest.setup.ts"],
    testMatch: ["**/__tests__/**/*.test.ts?(x)"],
    transformIgnorePatterns: [
        "node_modules/(?!(lucide-react|clsx|tailwind-merge|class-variance-authority)/)",
    ],
};

export default config;
