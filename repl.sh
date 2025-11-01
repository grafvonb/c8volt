#!/usr/bin/env python3
import argparse
import os
import re
from pathlib import Path
from typing import Iterable

PATTERN_WORD = re.compile(r"\bc8volt\b", re.IGNORECASE)  # whole word
PATTERN_ANY  = re.compile(r"c8volt", re.IGNORECASE)      # inside words
REPLACEMENT  = "c8volt"

def iter_files(root: Path) -> Iterable[Path]:
    for p in root.rglob("*"):
        if p.is_file() and not p.is_symlink():
            yield p

def is_probably_text(p: Path, sniff_bytes: int = 1024) -> bool:
    try:
        with p.open("rb") as f:
            block = f.read(sniff_bytes)
        if b"\x00" in block:
            return False
        block.decode("utf-8", errors="ignore")
        return True
    except Exception:
        return False

def main():
    ap = argparse.ArgumentParser(
        description="Replace any case of 'c8volt' with 'c8volt' recursively."
    )
    ap.add_argument("path", nargs="?", default=".", help="Root folder. Default: current directory")
    ap.add_argument("--dry-run", action="store_true", help="Show planned changes only")
    ap.add_argument("--include-hidden", action="store_true", help="Process hidden files and folders too")
    ap.add_argument("--anywhere", action="store_true", help="Match inside words, not just whole word")
    args = ap.parse_args()

    root = Path(args.path).resolve()
    pat = PATTERN_ANY if args.anywhere else PATTERN_WORD

    changed_files = 0
    total_replacements = 0

    for p in iter_files(root):
        if not args.include_hidden:
            parts = p.relative_to(root).parts
            if any(part.startswith(".") for part in parts):
                continue
        if not is_probably_text(p):
            continue

        try:
            text = p.read_text(encoding="utf-8", errors="ignore")
        except Exception:
            continue

        new_text, n = pat.subn(REPLACEMENT, text)
        if n == 0:
            continue

        if args.dry_run:
            print(f"[DRY] {p} -> {n} replacement(s)")
        else:
            try:
                p.write_text(new_text, encoding="utf-8")
                print(f"[OK]  {p} -> {n} replacement(s)")
            except Exception as e:
                print(f"[ERR] {p}: {e}")
                continue

        changed_files += 1
        total_replacements += n

    mode = "dry-run" if args.dry_run else "written"
    print(f"Summary: {total_replacements} replacement(s) across {changed_files} file(s) ({mode}).")

if __name__ == "__main__":
    main()

