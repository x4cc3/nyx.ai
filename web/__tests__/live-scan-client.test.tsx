import { render, screen } from "@testing-library/react";
import LiveScanClient from "@/app/scans/[scan_id]/LiveScanClient";
import { useFlowStream } from "@/lib/sse";

jest.mock("@/components/layout/AppShell", () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

jest.mock("next/link", () => ({
  __esModule: true,
  default: ({
    children,
    href,
    ...props
  }: {
    children: React.ReactNode;
    href: string;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

jest.mock("@/lib/sse", () => ({
  useFlowStream: jest.fn(),
}));

const mockUseFlowStream = useFlowStream as jest.MockedFunction<typeof useFlowStream>;

describe("LiveScanClient", () => {
  it("shows executor network status and warning when docker networking is enabled", () => {
    mockUseFlowStream.mockReturnValue({
      workspace: {
        flow: {
          id: "flow-1",
          tenant_id: "alpha",
          name: "Acme flow",
          target: "https://app.example.com",
          objective: "Inspect the target",
          status: "running",
          created_at: "2026-03-27T10:00:00Z",
          updated_at: "2026-03-27T10:00:00Z",
        },
        tasks: [],
        subtasks: [],
        actions: [],
        artifacts: [
          {
            id: "artifact-1",
            flow_id: "flow-1",
            action_id: "action-1",
            kind: "execution",
            name: "terminal_exec-trace",
            content: "command: curl https://app.example.com/healthz",
            metadata: {
              image: "nyx-executor-pentest:latest",
              network_mode: "custom",
              command: "curl https://app.example.com/healthz",
              evidence_paths: "/tmp/evidence.txt",
            },
            created_at: "2026-03-27T10:01:00Z",
          },
        ],
        memories: [],
        findings: [],
        agents: [],
        executions: [
          {
            id: "exec-1",
            flow_id: "flow-1",
            action_id: "action-1",
            profile: "terminal",
            runtime: "docker",
            metadata: {
              image: "nyx-executor-pentest:latest",
              network_mode: "custom",
              network_name: "nyx-targets",
              command: "curl https://app.example.com/healthz",
              evidence_paths: "/tmp/evidence.txt",
            },
            status: "completed",
            started_at: "2026-03-27T10:01:00Z",
            completed_at: "2026-03-27T10:02:00Z",
          },
        ],
        approvals: [],
        functions: [],
        tenant_id: "alpha",
        queue_mode: "jetstream",
        executor_mode: "docker",
        browser_mode: "http",
        executor_network_mode: "custom",
        executor_network_name: "nyx-targets",
        executor_net_raw_enabled: true,
        terminal_network_enabled: true,
        browser_warning:
          "Browser service is running in HTTP mode. Rendered screenshots and JavaScript execution require chromedp mode.",
        network_warning:
          'Networked terminal execution is enabled in docker mode on network "nyx-targets". Terminal scope checks still run before execution.',
        needs_review: false,
      },
      events: [],
      loading: false,
      error: null,
      connected: true,
    });

    render(<LiveScanClient scanId="flow-1" />);

    expect(screen.getByText("exec docker")).toBeInTheDocument();
    expect(screen.getByText("browser http")).toBeInTheDocument();
    expect(screen.getByText("net nyx-targets")).toBeInTheDocument();
    expect(screen.getByText("NET_RAW available")).toBeInTheDocument();
    expect(
      screen.getByText(/rendered screenshots and javascript execution require chromedp/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/scope checks still run before execution/i),
    ).toBeInTheDocument();
    expect(screen.getByText("Executions")).toBeInTheDocument();
    expect(screen.getAllByText(/nyx-executor-pentest:latest/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/curl https:\/\/app\.example\.com\/healthz/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/\/tmp\/evidence\.txt/i).length).toBeGreaterThan(0);
  });
});
