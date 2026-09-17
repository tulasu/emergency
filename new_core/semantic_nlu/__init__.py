# -*- coding: utf-8 -*-
"""Прототип semantic NLU рядом с trainer. Не подключён к Engine."""
try:
    from .understand import MatchResult, SemanticUnderstander
except ImportError:
    from understand import MatchResult, SemanticUnderstander

__all__ = ["MatchResult", "SemanticUnderstander"]
