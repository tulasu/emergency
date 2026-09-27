# -*- coding: utf-8 -*-
"""Парное сравнение не должно молча принять разные blind-выборки."""

import json

import pytest

from bench.pair import compare


def test_rejects_misaligned_blind_rows(tmp_path):
    left, right = tmp_path / "left.jsonl", tmp_path / "right.jsonl"
    sample = {"scenario": "bilet04_call01", "index": 0,
              "text": "Где вы?", "outcome": "ответ", "reveal": ["локация"]}
    left.write_text(json.dumps(sample) + "\n", encoding="utf-8")
    right.write_text(json.dumps({**sample, "text": "Другой вопрос"}) + "\n",
                     encoding="utf-8")
    with pytest.raises(ValueError, match="непарные входные реплики"):
        compare(left, right)
