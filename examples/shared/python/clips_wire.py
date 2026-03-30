"""
Local mirrors of CLIPS **provider chat** JSON (ClipsInput / ClipsOutput).

These are not exported by nxuskit-py. Use with json.dumps(...) into the user
message for provider chat, and json.loads on response.content.

Canonical reference: conformance/clips-json-contract.json (repository root).
SDK docs: sdk-packaging/docs/rule-authoring.md — ClipsInput JSON Reference.

For the Session API, use nxuskit.clips.ClipsSession instead of provider chat.

Helpers ``clips_input_to_json`` / ``clips_output_from_json`` round-trip provider-chat
message bodies using the same field names as Go/Rust ``clips*Wire`` types.
"""

from __future__ import annotations

import json
from dataclasses import asdict, dataclass, field
from typing import Any, Dict, List, Optional, Union


@dataclass
class ClipsFactWire:
    template: str
    values: Dict[str, Any]
    id: Optional[str] = None


@dataclass
class ClipsSlotWire:
    name: str
    slot_type: str  # JSON field name is "type" (CLIPS slot type: STRING, SYMBOL, …)


@dataclass
class ClipsTemplateWire:
    name: str
    slots: List[ClipsSlotWire] = field(default_factory=list)


@dataclass
class ClipsRequestConfigWire:
    max_rules: Optional[int] = None
    include_trace: Optional[bool] = None
    derived_only_new: Optional[bool] = None
    output_templates: List[str] = field(default_factory=list)
    stream_mode: Optional[str] = None


@dataclass
class ClipsInputWire:
    facts: List[ClipsFactWire] = field(default_factory=list)
    templates: List[ClipsTemplateWire] = field(default_factory=list)
    config: Optional[ClipsRequestConfigWire] = None
    focus: List[str] = field(default_factory=list)


@dataclass
class ClipsConclusionWire:
    template: str
    values: Dict[str, Any]
    fact_index: int = 0
    derived: bool = False
    id: Optional[str] = None


@dataclass
class ClipsExecStatsWire:
    total_rules_fired: int = 0
    conclusions_count: int = 0
    execution_time_ms: int = 0


@dataclass
class ClipsRuleFiringWire:
    rule_name: str
    fire_count: int = 0
    module: Optional[str] = None
    salience: int = 0


@dataclass
class ClipsTraceWire:
    rules_fired: List[ClipsRuleFiringWire] = field(default_factory=list)


@dataclass
class ClipsOutputWire:
    conclusions: List[ClipsConclusionWire] = field(default_factory=list)
    input_facts: List[ClipsConclusionWire] = field(default_factory=list)
    stats: ClipsExecStatsWire = field(default_factory=ClipsExecStatsWire)
    trace: Optional[ClipsTraceWire] = None


def _rename_slot_type_to_json_type(obj: Any) -> Any:
    """Recursively rename dataclass field ``slot_type`` to JSON key ``type`` (matches Go/Rust wire)."""
    if isinstance(obj, dict):
        out: Dict[str, Any] = {}
        for k, v in obj.items():
            if k == "slot_type":
                out["type"] = _rename_slot_type_to_json_type(v)
            else:
                out[k] = _rename_slot_type_to_json_type(v)
        return out
    if isinstance(obj, list):
        return [_rename_slot_type_to_json_type(x) for x in obj]
    return obj


def _prune_empty_optional_fields(d: Dict[str, Any]) -> Dict[str, Any]:
    """Match Go/Rust omitempty: drop empty templates, focus, and absent config."""
    out: Dict[str, Any] = {"facts": d.get("facts") or []}
    templates = d.get("templates") or []
    if templates:
        out["templates"] = templates
    cfg = d.get("config")
    if cfg is not None:
        out["config"] = cfg
    focus = d.get("focus") or []
    if focus:
        out["focus"] = focus
    return out


def clips_input_to_json(inp: ClipsInputWire, *, indent: Optional[int] = None) -> str:
    """Serialize ``ClipsInputWire`` to ClipsInput JSON for provider-chat user messages."""
    raw = asdict(inp)
    raw = _rename_slot_type_to_json_type(raw)
    raw = _prune_empty_optional_fields(raw)
    return json.dumps(raw, indent=indent, ensure_ascii=False)


def clips_output_from_json(s: Union[str, bytes]) -> ClipsOutputWire:
    """Parse CLIPS provider ``response.content`` into ``ClipsOutputWire``."""
    data = json.loads(s)

    def conclusion_from(c: Dict[str, Any]) -> ClipsConclusionWire:
        return ClipsConclusionWire(
            template=str(c.get("template", "")),
            values=dict(c.get("values") or {}),
            fact_index=int(c.get("fact_index", 0)),
            derived=bool(c.get("derived", False)),
            id=c.get("id"),
        )

    conclusions = [conclusion_from(c) for c in (data.get("conclusions") or [])]
    input_facts = [conclusion_from(c) for c in (data.get("input_facts") or [])]

    st = data.get("stats") or {}
    stats = ClipsExecStatsWire(
        total_rules_fired=int(st.get("total_rules_fired", 0)),
        conclusions_count=int(st.get("conclusions_count", 0)),
        execution_time_ms=int(st.get("execution_time_ms", 0)),
    )

    trace: Optional[ClipsTraceWire] = None
    tr = data.get("trace")
    if isinstance(tr, dict) and tr.get("rules_fired") is not None:
        fired = []
        for r in tr.get("rules_fired") or []:
            fired.append(
                ClipsRuleFiringWire(
                    rule_name=str(r.get("rule_name", "")),
                    fire_count=int(r.get("fire_count", 0)),
                    module=r.get("module"),
                    salience=int(r.get("salience", 0)),
                )
            )
        trace = ClipsTraceWire(rules_fired=fired)

    return ClipsOutputWire(
        conclusions=conclusions,
        input_facts=input_facts,
        stats=stats,
        trace=trace,
    )
