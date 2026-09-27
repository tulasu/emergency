# -*- coding: utf-8 -*-
"""Text normalization shared by live and offline Silero synthesis."""

from __future__ import annotations

import re

_ONES = ("ноль один два три четыре пять шесть семь восемь девять десять "
         "одиннадцать двенадцать тринадцать четырнадцать пятнадцать "
         "шестнадцать семнадцать восемнадцать девятнадцать").split()
_TENS = "_ _ двадцать тридцать сорок пятьдесят шестьдесят семьдесят восемьдесят девяносто".split()
_HUNDREDS = ("_ сто двести триста четыреста пятьсот шестьсот семьсот "
             "восемьсот девятьсот").split()


def _say999(n: int) -> str:
    """0..999 словами (м. р., им. п.)."""
    if n < 20:
        return _ONES[n]
    words = []
    if n >= 100:
        words.append(_HUNDREDS[n // 100])
        n %= 100
    if n >= 20:
        words.append(_TENS[n // 10])
        n %= 10
    if n:
        words.append(_ONES[n])
    return " ".join(words)


def _say_thousands(n: int) -> str:
    k, r = divmod(n, 1000)
    head = {1: "одна", 2: "две"}.get(k % 10) if k % 100 not in (11, 12) else None
    kw = _say999(k)
    if k == 1:
        return "тысяча" + (f" {_say999(r)}" if r else "")
    if head:
        kw = kw.rsplit(" ", 1)[0] + " " + head if " " in kw else head
    if k % 10 == 1 and k % 100 != 11:
        form = "тысяча"
    elif k % 10 in (2, 3, 4) and k % 100 not in (12, 13, 14):
        form = "тысячи"
    else:
        form = "тысяч"
    return f"{kw} {form}" + (f" {_say999(r)}" if r else "")


def _say_digits(m: re.Match) -> str:
    d = m.group(0)
    if len(d) >= 5:  # телефон/код: как диктуют — 3-3-2-2, ведущие нули цифрами
        groups, rest = [], d
        while len(rest) > 4:
            groups.append(rest[:3])
            rest = rest[3:]
        groups += [rest[:2], rest[2:]] if len(rest) == 4 else [rest]
    elif len(d) == 4 and d[0] == "0":  # код «0453» — парами
        groups = [d[:2], d[2:]]
    else:
        groups = [d]
    out = []
    for g in groups:
        lead = len(g) - len(g.lstrip("0"))
        out += ["ноль"] * min(lead, len(g) - 1)
        n = int(g)
        out.append(_say999(n) if n < 1000 else _say_thousands(n))
    return ", ".join(out)


def speakable(text: str) -> str:
    """Silero does not pronounce digits, so replace them with spoken Russian."""
    # «916 896 3254», «903-226-13-83» — one number before grouping.
    text = re.sub(r"\d[\d \-]{5,}\d",
                  lambda m: re.sub(r"[ \-]", "", m.group(0))
                  if sum(c.isdigit() for c in m.group(0)) >= 7 else m.group(0),
                  text)
    return re.sub(r"\d+", _say_digits, text)
