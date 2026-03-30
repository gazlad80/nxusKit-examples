//go:build nxuskit

package arbiter

// clips_*Wire types are local mirrors of the CLIPS provider chat JSON contract
// (ClipsInput / ClipsOutput in nxuskit-engine). They are not exported by nxuskit-go.
//
// Canonical reference: conformance/clips-json-contract.json (repo root).
// SDK documentation: nxusKit bundle sdk-packaging/docs/rule-authoring.md — ClipsInput JSON Reference.
//
// This example uses provider chat (JSON in the user message). For the Session API
// (direct engine), use nxuskit.ClipsSession instead.

type clipsFactWire struct {
	Template string                 `json:"template"`
	Values   map[string]interface{} `json:"values"`
	ID       *string                `json:"id,omitempty"`
}

type clipsSlotWire struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type clipsTemplateWire struct {
	Name  string          `json:"name"`
	Slots []clipsSlotWire `json:"slots"`
}

type clipsRequestConfigWire struct {
	MaxRules        *int64   `json:"max_rules,omitempty"`
	IncludeTrace    *bool    `json:"include_trace,omitempty"`
	DerivedOnlyNew  *bool    `json:"derived_only_new,omitempty"`
	OutputTemplates []string `json:"output_templates,omitempty"`
	StreamMode      *string  `json:"stream_mode,omitempty"`
}

type clipsInputWire struct {
	Facts     []clipsFactWire         `json:"facts"`
	Templates []clipsTemplateWire     `json:"templates,omitempty"`
	Config    *clipsRequestConfigWire `json:"config,omitempty"`
	Focus     []string                `json:"focus,omitempty"`
}

type clipsConclusionWire struct {
	Template  string                 `json:"template"`
	Values    map[string]interface{} `json:"values"`
	FactIndex int64                  `json:"fact_index"`
	Derived   bool                   `json:"derived"`
	ID        *string                `json:"id,omitempty"`
}

type clipsExecStatsWire struct {
	TotalRulesFired  uint64 `json:"total_rules_fired"`
	ConclusionsCount uint64 `json:"conclusions_count"`
	ExecutionTimeMs  uint64 `json:"execution_time_ms"`
}

type clipsRuleFiringWire struct {
	RuleName  string `json:"rule_name"`
	FireCount uint64 `json:"fire_count"`
	Module    string `json:"module,omitempty"`
	Salience  int32  `json:"salience,omitempty"`
}

type clipsTraceWire struct {
	RulesFired []clipsRuleFiringWire `json:"rules_fired"`
}

type clipsOutputWire struct {
	Conclusions []clipsConclusionWire `json:"conclusions"`
	InputFacts  []clipsConclusionWire `json:"input_facts,omitempty"`
	Stats       clipsExecStatsWire    `json:"stats"`
	Trace       *clipsTraceWire       `json:"trace,omitempty"`
}
