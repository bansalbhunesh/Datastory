import type { GenerateReportResponse } from "../types/report";

// Toggle this in `App.tsx` if OpenMetadata / Claude are flaky during a demo.
export const sampleReport: GenerateReportResponse = {
  source: "deterministic",
  warnings: ["Mock dataset — not connected to OpenMetadata."],
  tableFQN: "sample_data.ecommerce_db.shopify.fact_order",
  markdown: [
    "## Incident summary",
    "Order facts show elevated NULL rates in `revenue_usd` starting at 09:15 UTC. Downstream revenue dashboards are impacted.",
    "",
    "## Root cause analysis",
    "- Upstream `stg_orders` introduced a schema change that dropped `revenue_usd`.",
    "- Lineage shows the column is required by `fact_order` and three downstream dashboards.",
    "- A failing `columnValuesToBeNotNull` test confirms the regression scope.",
    "",
    "## Impact assessment",
    "- Downstream: Executive Revenue Dashboard, Marketing ROI model, Finance close checklist.",
    "",
    "## Severity",
    "**HIGH** — revenue reporting is materially wrong for multiple teams.",
    "",
    "## Recommended remediation",
    "- Restore `revenue_usd` in `stg_orders` transformation logic.",
    "- Backfill affected partitions after deploy.",
    "- Add a contract test on the staging model before promotion.",
  ].join("\n"),
  lineage: {
    focal: "sample_data.ecommerce_db.shopify.fact_order",
    upstream: [
      "sample_data.ecommerce_db.shopify.stg_orders",
      "sample_data.ecommerce_db.shopify.raw_shopify_orders",
    ],
    downstream: [
      "sample_data.ecommerce_db.shopify.dim_customer",
      "sample_data.ecommerce_db.shopify.report_daily_revenue",
    ],
    upstreamRaw: 6,
    downstreamRaw: 4,
  },
  failedTests: [
    {
      name: "fact_order.revenue_usd_not_null",
      status: "Failed",
      result: "Null fraction 0.18 exceeds threshold 0.00",
    },
  ],
};
