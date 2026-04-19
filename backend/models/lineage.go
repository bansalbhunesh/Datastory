package models

// LineageSummary is a lightweight view for UI + prompts.
type LineageSummary struct {
	Focal         string   `json:"focal"`
	Upstream      []string `json:"upstream"`
	Downstream    []string `json:"downstream"`
	UpstreamRaw   int      `json:"upstreamRaw"`
	DownstreamRaw int      `json:"downstreamRaw"`
}
