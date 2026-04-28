import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { SeverityBadge } from "./SeverityBadge";

describe("SeverityBadge", () => {
  test.each(["LOW", "MEDIUM", "HIGH", "UNKNOWN"] as const)(
    "renders %s text",
    (sev) => {
      render(<SeverityBadge severity={sev} />);
      expect(screen.getByText(`Severity: ${sev}`)).toBeInTheDocument();
    },
  );
});
