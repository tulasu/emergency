"""Pure ticketgen CLI: prompt -> draft JSON on stdout.

No DB, no claim/lease loop, no ticketgen<->dialog link. Traineebox owns
job rows (building_dialog/checking_dialog) and the atomic approve txn;
this tool runs the pure Pipeline once (lint gate, debugging, seeding).

Usage:
  python -m worker --catalog <dir> --prompt "пожар на складе"
  echo "пожар" | python -m worker --catalog <dir>
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from catalog import load_catalog
from llm import LLMClient
from pipeline import Pipeline


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--catalog", required=True)
    ap.add_argument("--prompt", default="")
    args = ap.parse_args()

    prompt = args.prompt or sys.stdin.read().strip()
    if not prompt:
        raise SystemExit("need --prompt or stdin")
    catalog = load_catalog(Path(args.catalog))
    result = Pipeline(catalog, LLMClient()).draft_scenario(prompt)
    json.dump(
        {
            "draft_title": result.draft_title,
            "scenario_text": result.scenario_text,
            "draft_reference": result.draft_reference,
        },
        sys.stdout,
        ensure_ascii=False,
        indent=2,
    )
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
