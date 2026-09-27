# -*- coding: utf-8 -*-
"""Нормализация: числа, леммы, сегменты."""

import pytest

from dispatcher.nlu import normalize


@pytest.mark.parametrize(
    "text,expected",
    [
        ("тридцать семь", 37),
        ("сто двадцать три", 123),
        ("сорок пять", 45),
        ("девяносто девять", 99),
    ],
)
def test_compound_numerals_summed(text, expected):
    """Оператор зачитывает номер словами, а в билете он цифрой."""
    assert expected in normalize.numbers(text)


def test_compound_parts_kept_too():
    """«Тридцать седьмой» и «тридцать» оба могут оказаться значением факта."""
    n = normalize.numbers("тридцать семь")
    assert {30, 7, 37} <= n


def test_digits_parsed():
    assert 37 in normalize.numbers("дом 37")
    assert 13 in normalize.numbers("на тринадцатом этаже")


def test_phone_not_summed_to_one_number():
    """«Три пять семь» — это цифры номера, а не триста пятьдесят семь."""
    n = normalize.numbers("три пять семь")
    assert n == {3, 5, 7}


def test_number_after_word_not_glued():
    n = normalize.numbers("двадцать этажей и семь подъездов")
    assert 27 not in n and {20, 7} <= n


def test_question_words_scored_not_topical():
    assert "что" in normalize.content_lemmas("что случилось")
    assert "что" not in normalize.topical_lemmas("что случилось")


def test_pronouns_tell_caller_from_victim():
    assert normalize.content_lemmas("как вас зовут") != normalize.content_lemmas(
        "как её зовут"
    )


def test_hyphenated_words_kept():
    assert "ваз" in normalize.words("ВАЗ-2110")
