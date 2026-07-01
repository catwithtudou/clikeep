#!/usr/bin/env python3
"""Lightweight validation for repository skills.

This intentionally avoids non-stdlib dependencies so it can run in CI before any
project-specific Python environment is prepared.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path


NAME_RE = re.compile(r"^[a-z0-9-]+$")
MAX_NAME_LENGTH = 64
MAX_DESCRIPTION_LENGTH = 1024


class ValidationError(Exception):
    pass


def parse_simple_yaml_mapping(text: str, source: Path) -> dict[str, str]:
    data: dict[str, str] = {}
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if raw_line.startswith((" ", "\t")):
            continue
        if ":" not in line:
            raise ValidationError(f"{source}: invalid YAML line: {raw_line!r}")
        key, value = line.split(":", 1)
        key = key.strip()
        value = value.strip()
        if value.startswith('"') and value.endswith('"'):
            value = value[1:-1]
        elif value.startswith("'") and value.endswith("'"):
            value = value[1:-1]
        data[key] = value
    return data


def parse_frontmatter(skill_md: Path) -> dict[str, str]:
    content = skill_md.read_text(encoding="utf-8")
    if not content.startswith("---\n"):
        raise ValidationError(f"{skill_md}: missing YAML frontmatter")
    end = content.find("\n---", 4)
    if end == -1:
        raise ValidationError(f"{skill_md}: unterminated YAML frontmatter")
    frontmatter = content[4:end]
    data = parse_simple_yaml_mapping(frontmatter, skill_md)
    unexpected = set(data) - {"name", "description"}
    if unexpected:
        keys = ", ".join(sorted(unexpected))
        raise ValidationError(f"{skill_md}: unexpected frontmatter keys: {keys}")
    return data


def parse_openai_yaml(openai_yaml: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw_line in openai_yaml.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#") or ":" not in line:
            continue
        key, value = line.split(":", 1)
        key = key.strip()
        value = value.strip()
        if value.startswith('"') and value.endswith('"'):
            value = value[1:-1]
        values[key] = value
    return values


def validate_skill(skill_dir: Path) -> None:
    skill_md = skill_dir / "SKILL.md"
    if not skill_md.exists():
        raise ValidationError(f"{skill_dir}: SKILL.md not found")

    frontmatter = parse_frontmatter(skill_md)
    name = frontmatter.get("name", "").strip()
    description = frontmatter.get("description", "").strip()

    if not name:
        raise ValidationError(f"{skill_md}: missing name")
    if not NAME_RE.match(name):
        raise ValidationError(f"{skill_md}: name must be hyphen-case")
    if name.startswith("-") or name.endswith("-") or "--" in name:
        raise ValidationError(f"{skill_md}: invalid hyphen placement in name")
    if len(name) > MAX_NAME_LENGTH:
        raise ValidationError(f"{skill_md}: name is too long")
    if skill_dir.name != name:
        raise ValidationError(f"{skill_dir}: directory name must match skill name {name!r}")

    if not description:
        raise ValidationError(f"{skill_md}: missing description")
    if "<" in description or ">" in description:
        raise ValidationError(f"{skill_md}: description must not contain angle brackets")
    if len(description) > MAX_DESCRIPTION_LENGTH:
        raise ValidationError(f"{skill_md}: description is too long")

    openai_yaml = skill_dir / "agents" / "openai.yaml"
    if not openai_yaml.exists():
        raise ValidationError(f"{skill_dir}: agents/openai.yaml not found")
    values = parse_openai_yaml(openai_yaml)
    display_name = values.get("display_name", "").strip()
    short_description = values.get("short_description", "").strip()
    default_prompt = values.get("default_prompt", "").strip()

    if not display_name:
        raise ValidationError(f"{openai_yaml}: missing interface.display_name")
    if not (25 <= len(short_description) <= 64):
        raise ValidationError(f"{openai_yaml}: short_description must be 25-64 chars")
    if f"${name}" not in default_prompt:
        raise ValidationError(f"{openai_yaml}: default_prompt must mention ${name}")


def find_skill_dirs(root: Path) -> list[Path]:
    if not root.exists():
        raise ValidationError(f"{root}: path not found")
    if (root / "SKILL.md").exists():
        return [root]
    return sorted(path for path in root.iterdir() if (path / "SKILL.md").exists())


def main(argv: list[str]) -> int:
    if len(argv) != 2:
        print("usage: validate_skills.py <skills-dir-or-skill>", file=sys.stderr)
        return 2
    root = Path(argv[1])
    try:
        skill_dirs = find_skill_dirs(root)
        if not skill_dirs:
            raise ValidationError(f"{root}: no skills found")
        for skill_dir in skill_dirs:
            validate_skill(skill_dir)
            print(f"ok {skill_dir}")
    except ValidationError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
