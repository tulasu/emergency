"""audio-synth contract tests: stdlib only, no torch required."""
import io
import shutil
import sys
import unittest
import wave
from pathlib import Path

HERE = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(HERE.parent))

from dialog.core.media.speakable import speakable  # noqa: E402

sys.path.insert(0, str(HERE))


class SpeakableTest(unittest.TestCase):
    def test_numbers_are_spoken_for_synthesis(self):
        self.assertEqual(speakable("дом 19"), "дом девятнадцать")
        self.assertEqual(speakable("код 0453"), "код ноль, четыре, пятьдесят три")

    def test_phone_runs_are_grouped_before_being_spoken(self):
        self.assertEqual(
            speakable("8 916 896 3254"),
            "восемьсот девяносто один, шестьсот восемьдесят девять, "
            "шестьсот тридцать два, пятьдесят четыре",
        )


class AppContractTest(unittest.TestCase):
    def test_voice_allowlist(self):
        import app

        self.assertIn("kseniya", app.VOICES)
        self.assertEqual(app.OUT_RATE, 8000)

    @unittest.skipUnless(shutil.which("ffmpeg"), "ffmpeg required")
    def test_neural_russian_fallback_produces_sized_pcm_wav(self):
        import app

        if not Path(app.PIPER_MODEL_PATH).is_file():
            self.skipTest("Piper Russian model not installed")
        from piper import PiperVoice

        app._piper = PiperVoice.load(app.PIPER_MODEL_PATH)
        data = app.piper_wav("Алло. Подскажите ваш адрес.")
        with wave.open(io.BytesIO(data), "rb") as wav:
            self.assertEqual((wav.getnchannels(), wav.getframerate(),
                              wav.getsampwidth()), (1, 8000, 2))
            self.assertGreater(wav.getnframes(), 0)
            self.assertEqual(len(data), 44 + wav.getnframes() * 2)


if __name__ == "__main__":
    unittest.main()
