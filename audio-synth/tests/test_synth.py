"""audio-synth contract tests: stdlib only, no torch required.

speakable parity is checked against dispatcher/tools/synth_audio.py by
exec of the pure function slice (no dispatcher import, no deps).
"""
import importlib.util
import sys
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(HERE))

import speakable as ours  # noqa: E402


def _load_dispatcher_speakable():
    path = HERE.parent / "dispatcher" / "tools" / "synth_audio.py"
    if not path.exists():
        # worktree without dispatcher history — parity checked in main repo CI
        return None
    src = path.read_text(encoding="utf-8")
    start = src.index("_ONES = ")
    end = src.index("def speakable(")
    end = src.index("return re.sub(r\"\\d+\"", end) + len("return re.sub(r\"\\d+\"") + len(", _say_digits, text)")
    ns: dict = {"re": __import__("re")}
    exec(src[start:end], ns)
    return ns["speakable"]


class SpeakableTest(unittest.TestCase):
    def test_digits(self):
        self.assertEqual(ours.speakable("дом 19"), "дом девятнадцать")
        # spaced digit runs glue into one number first (phone rule)
        self.assertEqual(ours.speakable("8 916 896 3254"),
                         "восемьсот девяносто один, шестьсот восемьдесят девять, "
                         "шестьсот тридцать два, пятьдесят четыре")

    def test_parity_with_dispatcher(self):
        ref = _load_dispatcher_speakable()
        if ref is None:
            self.skipTest("no dispatcher/tools/synth_audio.py in worktree")
        cases = ["дом 19", "8 916 896 3254", "код 0453", "Ленина 5, кв 1234567", "без цифр"]
        for c in cases:
            self.assertEqual(ours.speakable(c), ref(c), f"drift on {c!r}")


class AppContractTest(unittest.TestCase):
    def test_voice_allowlist(self):
        import app

        self.assertIn("kseniya", app.VOICES)
        self.assertEqual(app.OUT_RATE, 8000)


if __name__ == "__main__":
    unittest.main()
