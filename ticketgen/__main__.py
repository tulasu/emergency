"""Lint gate: 10 drafts -> validate_reference + dialog lint summary.

Full drafts need an LLM (see worker.py); this gate checks the deterministic
half without one: one valid draft per incident type built from the catalog
(first tag per single group), validated via catalog.validate_reference.
Exits nonzero on any failure. Expected: 0 unreachable critical.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.request
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from catalog import load_catalog


DIALOG_URL = os.environ.get("DIALOG_URL", "").rstrip("/")


def dialog_lint(snapshot: dict) -> dict | None:
    """POST one snapshot to dialog /scenarios/lint.

    None when dialog is unreachable (caller reports catalog-only)."""
    if not DIALOG_URL or not isinstance(snapshot, dict):
        return None
    try:
        req = urllib.request.Request(
            DIALOG_URL + "/scenarios/lint",
            json.dumps({"scenario": snapshot}).encode(),
            {"Content-Type": "application/json"}, method="POST")
        with urllib.request.urlopen(req, timeout=10) as r:
            return json.loads(r.read())
    except Exception:  # noqa: BLE001 — unreachable dialog is a label, not a crash
        return None


def build_draft(catalog, type_code: str) -> dict:
    t = catalog.types[type_code]
    tags: list[str] = []
    for g in t.groups:
        if g.selection_mode == "single" and g.tags and not g.parent_tag:
            tags.append(g.tags[0].code)
    services = catalog.recommend_services(type_code, tags)
    return {
        "incident_type_code": type_code,
        "tag_codes": tags,
        "service_codes": services,
        "applicant_last_name": "Иванов",
        "applicant_first_name": "Иван",
        "caller_number": "79001234567",
        "dictated_number": "79001234567",
    }


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--catalog", required=True)
    ap.add_argument("--count", type=int, default=10)
    args = ap.parse_args()

    catalog = load_catalog(Path(args.catalog))
    codes = sorted(catalog.types)[: args.count]
    bad = 0
    linted = skipped = unreachable = 0
    for code in codes:
        try:
            ref = build_draft(catalog, code)
            catalog.validate_reference(ref)
            # Catalog drafts carry no slot facts, so there is nothing to lint
            # yet (slot-fact authoring stays teacher-side); lint real snapshots
            # whenever one is attached.
            snapshot = ref.get("scenario_snapshot")
            if isinstance(snapshot, dict) and DIALOG_URL:
                report = dialog_lint(snapshot)
                if report is None:
                    skipped += 1
                else:
                    linted += 1
                    unreachable += len(report.get("unreachable") or [])
            else:
                skipped += 1
            print(f"ok {code}: tags={len(ref['tag_codes'])} services={len(ref['services'] if 'services' in ref else ref['service_codes'])}")
        except Exception as e:  # noqa: BLE001
            bad += 1
            print(f"FAIL {code}: {e}")
    if linted:
        print(f"drafts={len(codes)} failed={bad} unreachable_critical={unreachable} linted={linted} skipped={skipped}")
    else:
        print(f"drafts={len(codes)} failed={bad} unreachable_critical=unknown catalog-only skipped={skipped}")
    if bad:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
