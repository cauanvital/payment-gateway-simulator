#!/usr/bin/env python3
"""Merge Go coverage profiles, marking a block covered if any profile covers it."""

from __future__ import annotations

import sys
from pathlib import Path


def read_profile(path: Path) -> tuple[str, dict[tuple[str, str], int]]:
    lines = path.read_text(encoding="utf-8").splitlines()
    if not lines or not lines[0].startswith("mode: "):
        raise ValueError(f"{path}: invalid Go coverage profile")

    mode = lines[0]
    blocks: dict[tuple[str, str], int] = {}
    for line in lines[1:]:
        location, statements, count = line.rsplit(" ", 2)
        key = (location, statements)
        blocks[key] = max(blocks.get(key, 0), int(count))
    return mode, blocks


def main() -> None:
    if len(sys.argv) < 4:
        raise SystemExit(
            f"usage: {Path(sys.argv[0]).name} OUTPUT_PROFILE PROFILE [PROFILE ...]"
        )

    output_path = Path(sys.argv[1])
    mode: str | None = None
    merged: dict[tuple[str, str], int] = {}
    for argument in sys.argv[2:]:
        profile_mode, blocks = read_profile(Path(argument))
        if mode is not None and profile_mode != mode:
            raise SystemExit("coverage profiles use different modes")
        mode = profile_mode
        for key, count in blocks.items():
            merged[key] = max(merged.get(key, 0), count)

    output = [mode]
    for (location, statements), count in sorted(merged.items()):
        output.append(f"{location} {statements} {count}")
    output_path.write_text("\n".join(output) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
