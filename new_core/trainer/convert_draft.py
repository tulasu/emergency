# -*- coding: utf-8 -*-
"""Черновик по PROMPT_scenario_draft.md -> сценарий тренажёра.

  python3 convert_draft.py drafts/draft_bilet01_call01.json -o scenarios/bilet01_call01.json

Маппинг: paraphrases -> questions; replies.ask -> answers.plain,
repeat -> short, confirm -> confirm, correct -> correct.
replies.assert никуда не маппится: движок стиля ASSERT не имеет
(несогласие с числом — это Style.CORRECT), такой ответ в сценарии мёртвый.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


def convert(draft: dict) -> dict:
    facts = []
    for f in draft["facts"]:
        r = f.get("replies", {})
        if "ask" not in r:
            sys.exit(f"{f['key']}: нет обязательной реплики ask")
        answers = {"plain": r["ask"]}
        if r.get("repeat"):
            answers["short"] = r["repeat"]
        if r.get("confirm"):
            answers["confirm"] = r["confirm"]
        if r.get("correct"):
            answers["correct"] = r["correct"]
        if f.get("value_numbers") and "correct" not in answers:
            print(f"  ! {f['key']}: есть числа, но нет correct", file=sys.stderr)
        if len(f.get("paraphrases", [])) < 5:
            print(f"  ! {f['key']}: меньше 5 перефразировок", file=sys.stderr)
        facts.append({
            "key": f["key"],
            "group": f.get("group", f["key"]),
            "value_numbers": f.get("value_numbers", []),
            **({"requires": f["requires"]} if f.get("requires") else {}),
            "questions": f["paraphrases"],
            "answers": answers,
        })
    for k in draft.get("critical", []):
        if k not in {f["key"] for f in facts}:
            sys.exit(f"critical ссылается на несуществующий факт: {k}")
    return {
        "id": draft["scenario_id"],
        "meta": draft.get("meta", {}),
        "persona": {
            "profile": draft.get("profile", "calm"),
            "opening": draft["opening"],
            "voice": draft.get("voice", "female_45"),
        },
        "critical": draft.get("critical", []),
        "facts": facts,
    }


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("draft")
    ap.add_argument("-o", "--out", required=True)
    a = ap.parse_args()
    draft = json.loads(Path(a.draft).read_text(encoding="utf-8"))
    out = convert(draft)
    Path(a.out).write_text(json.dumps(out, ensure_ascii=False, indent=1),
                           encoding="utf-8")
    print(f"{out['id']}: фактов {len(out['facts'])}, "
          f"critical {len(out['critical'])} -> {a.out}")


if __name__ == "__main__":
    main()
