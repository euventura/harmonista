#!/usr/bin/env python3
"""
Simple CSS minifier for this repo:
- produces public/css/base.slim.css (comments removed, collapsed blank lines)
- produces public/css/base.min.css (single-line, whitespace-collapsed)

Safe, non-ambitious minification: removes /*...*/ comments and collapses whitespace.
"""
import re
from pathlib import Path

root = Path(__file__).resolve().parents[1]
css_in = root / 'public' / 'css' / 'base.css'
css_slim = root / 'public' / 'css' / 'base.slim.css'
css_min = root / 'public' / 'css' / 'base.min.css'

if not css_in.exists():
    raise SystemExit(f"Input CSS not found: {css_in}")

text = css_in.read_text(encoding='utf-8')
# Remove all /* ... */ comments (non-greedy)
text_no_comments = re.sub(r'/\*.*?\*/', '', text, flags=re.DOTALL)

# Slim: collapse multiple blank lines to one, strip trailing spaces on lines
lines = [ln.rstrip() for ln in text_no_comments.splitlines()]
# remove leading/trailing blank lines
while lines and lines[0] == '':
    lines.pop(0)
while lines and lines[-1] == '':
    lines.pop()
# collapse multiple blank lines
out_lines = []
blank = 0
for ln in lines:
    if ln == '':
        blank += 1
    else:
        blank = 0
    if blank <= 1:
        out_lines.append(ln)
slim = '\n'.join(out_lines) + '\n'
css_slim.write_text(slim, encoding='utf-8')

# Minified: remove all newlines and collapse whitespace sequences to a single space
# Keep single spaces inside quoted strings by using a simple approach: we won't strip inside quotes explicitly,
# but collapsing whitespace globally is acceptable for this CSS content.
minified = ' '.join(slim.split())
# Optional: remove space before and after certain punctuation to shrink a bit
minified = re.sub(r'\s*([{}:;,>+~\(\)])\s*', r"\1", minified)
minified += '\n'
css_min.write_text(minified, encoding='utf-8')

print(f"Wrote: {css_slim}\nWrote: {css_min}")
print(f"Original size: {css_in.stat().st_size} bytes")
print(f"Slim size: {css_slim.stat().st_size} bytes")
print(f"Min size: {css_min.stat().st_size} bytes")
