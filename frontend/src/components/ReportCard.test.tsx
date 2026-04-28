import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import { ReportCard } from "./ReportCard";

const md = "## Incident summary\n\nbody text\n\n## Severity\n\n**HIGH** — heuristic.";

describe("ReportCard", () => {
  test("renders title, subtitle and source pill", () => {
    render(
      <ReportCard
        title="Incident report"
        subtitle="svc.db.s.t"
        markdown={md}
        source="claude"
      />,
    );
    expect(screen.getByText("Incident report")).toBeInTheDocument();
    expect(screen.getByText("svc.db.s.t")).toBeInTheDocument();
    expect(screen.getByText(/AI-Enhanced/i)).toBeInTheDocument();
  });

  test("rule-based pill when source=deterministic", () => {
    render(<ReportCard title="r" markdown={md} source="deterministic" />);
    expect(screen.getByText(/Rule-Based/i)).toBeInTheDocument();
  });

  test("copy button writes to clipboard and shows feedback", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });

    render(<ReportCard title="r" markdown={md} source="claude" />);
    await user.click(screen.getByRole("button", { name: /^Copy$/ }));

    expect(writeText).toHaveBeenCalledWith(md);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Copied/i })).toBeInTheDocument();
    });
  });

  test("copy failure surfaces a permission message", async () => {
    const user = userEvent.setup();
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: () => Promise.reject(new Error("denied")) },
      configurable: true,
    });

    render(<ReportCard title="r" markdown={md} source="claude" />);
    await user.click(screen.getByRole("button", { name: /^Copy$/ }));

    await waitFor(() => {
      // Both the visible amber message AND the sr-only live region announce
      // it, so we expect at least one match.
      expect(screen.getAllByText(/Clipboard permission denied/i).length).toBeGreaterThan(0);
    });
  });

  test("warnings are translated to user-friendly text", () => {
    render(
      <ReportCard
        title="r"
        markdown={md}
        source="deterministic"
        warnings={["AI enhancement not configured — showing verified rule-based report."]}
      />,
    );
    expect(
      screen.getByText(/AI enhancement not configured/i),
    ).toBeInTheDocument();
  });
});
