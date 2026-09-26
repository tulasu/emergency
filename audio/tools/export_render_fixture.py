# -*- coding: utf-8 -*-
"""Export dialog render constants to JSON for the Go parity test.

CI runs this against dialog/core/dialog/render.py and diffs the result with
audio/internal/enumerate/testdata/render_fixture.json. The Go service embeds
the same fixture as fallback; TestEnumerateParity fails on drift so dialog
wording changes land in audio deliberately, not silently.

Usage: python3 audio/tools/export_render_fixture.py > audio/internal/enumerate/testdata/render_fixture.json
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
RENDER = ROOT / "dialog" / "core" / "dialog" / "render.py"
TYPES = ROOT / "dialog" / "core" / "types.py"


def _load_module(name: str, path: Path):
    """Load one dialog module by path with stubbed sibling imports.

    render.py only needs Ontology (for the Renderer class body, not the
    GENERIC/SLOW_DOWN/URGE constants), so dialog.core.data.ontology is
    stubbed; dialog.core.types enums are loaded for real from types.py.
    No third-party deps (yaml/numpy/pymorphy3) are imported this way.
    """
    import importlib.util
    import types as pytypes

    pkg_root = pytypes.ModuleType("dialog_export_stub")
    core = pytypes.ModuleType("dialog.core")
    dialog_pkg = pytypes.ModuleType("dialog.core.dialog")
    data_pkg = pytypes.ModuleType("dialog.core.data")
    onto_mod = pytypes.ModuleType("dialog.core.data.ontology")

    class Ontology:  # noqa: D106 — stub, only Renderer.__init__ default needs it
        @classmethod
        def load(cls):
            raise NotImplementedError

    onto_mod.Ontology = Ontology
    sys.modules["dialog.core"] = core
    sys.modules["dialog.core.dialog"] = dialog_pkg
    sys.modules["dialog.core.data"] = data_pkg
    sys.modules["dialog.core.data.ontology"] = onto_mod

    spec_t = importlib.util.spec_from_file_location("dialog.core.types", TYPES)
    types_mod = importlib.util.module_from_spec(spec_t)
    sys.modules["dialog.core.types"] = types_mod
    spec_t.loader.exec_module(types_mod)
    core.types = types_mod

    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    sys.modules[name] = mod
    spec.loader.exec_module(mod)
    return mod


_render = _load_module("dialog.core.dialog.render", RENDER)
GENERIC, SLOW_DOWN, URGE = _render.GENERIC, _render.SLOW_DOWN, _render.URGE


def main() -> None:
    generic = {
        style.value: {mood.name.lower(): list(texts) for mood, texts in by_mood.items()}
        for style, by_mood in GENERIC.items()
    }
    slowdown = {mood.name.lower(): text for mood, text in SLOW_DOWN.items()}
    urge = {mood.name.lower(): text for mood, text in URGE.items()}
    json.dump(
        {"generic": generic, "slow_down": slowdown, "urge": urge},
        sys.stdout,
        ensure_ascii=False,
        indent=1,
        sort_keys=True,
    )
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
