#!/usr/bin/env python3
"""Dependency-free static CSS guard for AhB's embedded Playground UI."""
from pathlib import Path
import re
import sys


def check(source: str) -> None:
    match = re.search(r'<style>(.*?)</style>', source, flags=re.S)
    if not match:
        raise ValueError('Playground <style> block missing')
    css = match.group(1)
    stack = []
    quote = None
    comment = False
    escaped = False
    i = 0
    while i < len(css):
        c = css[i]
        nxt = css[i + 1] if i + 1 < len(css) else ''
        if comment:
            if c == '*' and nxt == '/':
                comment = False
                i += 2
                continue
        elif quote:
            if escaped:
                escaped = False
            elif c == '\\':
                escaped = True
            elif c == quote:
                quote = None
        elif c == '/' and nxt == '*':
            comment = True
            i += 2
            continue
        elif c in ('"', "'"):
            quote = c
        elif c == '{':
            stack.append(css.count('\n', 0, i) + 1)
        elif c == '}':
            if not stack:
                raise ValueError(f'extra CSS closing brace on line {css.count(chr(10), 0, i) + 1}')
            stack.pop()
        i += 1
    if comment or quote or stack:
        raise ValueError(f'unterminated CSS comment/quote/block (open block lines: {stack})')

    # iPhone Safari may zoom on focus when text controls use <16px type.
    selector = r'\.field input\s*,\s*\.field select\s*,\s*\.field textarea'
    rules = re.findall(selector + r'\s*\{([^{}]*)\}', css)
    if not rules:
        raise ValueError('Playground form-control CSS selector missing')
    sizes = [float(n) for rule in rules for n in re.findall(r'font-size\s*:\s*([0-9.]+)px', rule)]
    if not sizes or min(sizes) < 16:
        raise ValueError(f'Playground form inputs must use >=16px font (found {sizes})')
    for required in ('@media(hover:none) and (pointer:coarse)',
                     'prefers-reduced-motion:reduce', 'safe-area-inset-bottom',
                     ':focus-visible'):
        if required not in css:
            raise ValueError(f'Playground accessibility CSS missing: {required}')


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit('usage: python3 scripts/test-playground-css.py internal/hub/playground_ui.go')
    check(Path(sys.argv[1]).read_text(encoding='utf-8'))
    print('PASS: Playground CSS syntax, mobile input, touch and accessibility guards')


if __name__ == '__main__':
    main()
