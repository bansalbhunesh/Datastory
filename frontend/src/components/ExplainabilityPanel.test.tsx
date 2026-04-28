import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import type { GenerateReportResponse } from "../types/report";
import { ExplainabilityPanel } from "./ExplainabilityPanel";

function makeReport(overrides: Partial<GenerateReportResponse> = {}): GenerateReportResponse {
  return {
    tableFQN: "svc.db.s.t",
    markdown: "## x",
    severity: "HIGH",
    summary: "",
    rootCauses: [],
    impacts: [],
    remediation: [],
    lineage: { focal: "", upstream: [], downstream: [], upstreamRaw: 0, downstreamRaw: 0 },
    failedTests: [],
    explanation: {
      failedTestCount: 2,
      downstreamCount: 4,
      upstreamCount: 1,
      lineageComplete: true,
      confidence: 88,
    },
    source: "deterministic",
    warnings: [],
    ...overrides,
  };
}

describe("ExplainabilityPanel", () => {
  test("returns null without explanation", () => {
    const { container } = render(
      <ExplainabilityPanel report={makeReport({ explanation: undefined })} />,
    );
    expect(container.firstChild).toBeNull();
  });

  test("renders all severity factors and confidence", () => {
    render(<ExplainabilityPanel report={makeReport()} />);
    expect(screen.getByText("Failed quality tests")).toBeInTheDocument();
    expect(screen.getByText("Downstream tables at risk")).toBeInTheDocument();
    expect(screen.getByText("Upstream dependencies")).toBeInTheDocument();
    expect(screen.getByText("Lineage data")).toBeInTheDocument();
    expect(screen.getByText("88%")).toBeInTheDocument();
    expect(screen.getByText("HIGH")).toBeInTheDocument();
  });
});
