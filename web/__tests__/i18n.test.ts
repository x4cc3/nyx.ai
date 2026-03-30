import { translate } from "@/lib/i18n";

describe("translate", () => {
  it("returns the new report unknown-verification labels", () => {
    expect(translate("report.unknown.title")).toBe("Unknown verification");
    expect(translate("report.verification.unknown")).toBe("Unknown");
    expect(translate("report.verification.unknown_short")).toBe("UNK");
    expect(translate("severity.unknown")).toBe("unknown");
  });

  it("interpolates variables", () => {
    expect(translate("report.coverage.calls", { count: 7 })).toBe("7 calls");
    expect(translate("report.coverage.timed_out_tools")).toBe(
      "Timed-out tools",
    );
    expect(translate("report.coverage.unknown_time")).toBe("time unavailable");
  });
});
