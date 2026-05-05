#!/usr/bin/env python3
"""
Simple CSS minifier for this repo:
- produces public/css/base.min.css (single-line, whitespace-collapsed)

Safe, non-ambitious minification: removes /*...*/ comments and collapses whitespace.
"""
import re
from pathlib import Path

root = Path(__file__).resolve().parents[1]
css_in = root / 'public' / 'css' / 'base.css'
css_min = root / 'public' / 'css' / 'base.min.css'

if not css_in.exists():
    raise SystemExit(f"Input CSS not found: {css_in}")

text = css_in.read_text(encoding='utf-8')

# 1. Remove all /* ... */ comments (non-greedy)
text_no_comments = re.sub(r'/\*.*?\*/', '', text, flags=re.DOTALL)

# 2. Minified: remove all newlines and collapse whitespace sequences to a single space
minified = ' '.join(text_no_comments.split())

# 3. Optional: remove space before and after certain punctuation to shrink even more
minified = re.sub(r'\s*([{}:;,>+~\(\)])\s*', r"\1", minified)

# Add a single trailing newline for POSIX compliance
minified += '\n'

css_min.write_text(minified, encoding='utf-8')

print(f"Wrote: {css_min}")
print(f"Original size: {css_in.stat().st_size} bytes")
print(f"Minified size: {css_min.stat().st_size} bytes")
