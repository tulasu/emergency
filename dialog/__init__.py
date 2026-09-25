# -*- coding: utf-8 -*-
"""Stateless snapshot executor (dialog), owned by traineebox: shell over the frozen ML core."""
from .bank_source import Bank, digest_of
from .serve import main
from .validator import validate_snapshot

__all__ = ["Bank", "digest_of", "main", "validate_snapshot"]
