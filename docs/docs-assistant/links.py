#!/usr/bin/env python3

"""
Check Markdown links in documentation.

Usage:
  ./links.py <DOCS_SOURCE_PATH>
  ./links.py --grafana-hugo <DOCS_SOURCE_PATH>

**Tempo / Grafana Hugo docs:** always pass --grafana-hugo when checking
docs/sources/ (or run ./run-local-checks.sh). Otherwise /docs/ and other
site-root paths are reported as errors even though the site build resolves them.

Default mode expects relative in-repo links (Grafana internal docs style).

--grafana-hugo  For sources built with Hugo where /docs/, /media/, etc. are
                rewritten to https://grafana.com/... — skip those instead of
                reporting false positives. Still validates relative Markdown links
                against files on disk.
"""

import argparse
import os
import re
import sys
from pathlib import Path
from typing import List, Union

# Global list to collect error messages
errors: List[str] = []


def error(src: Union[str, Path], link: str, msg: str) -> None:
	"""
	Record an error message for a link in a source file.

	Args:
		src: Source file path where the error occurred.
		link: Problematic link string.
		msg: Descriptive error message.
	"""
	errors.append(f"ERROR: {src} - {msg}: {link}")


def skip_grafana_hugo_site_link(link: str) -> bool:
	"""
	Paths that resolve on the Grafana website; not checkable as local files.

	Hugo prepends https://grafana.com for site-root paths like /docs/...;
	ref: is a shortcode; full grafana.com URLs are skipped earlier.
	"""
	if link.startswith("/docs/"):
		return True
	if link.startswith("/media/"):
		return True
	if link.startswith("/static/"):
		return True
	if link.startswith("/blog/"):
		return True
	if link.startswith("ref:"):
		return True
	return False


def main() -> None:
	"""
	Main execution function.

	Parse command line arguments, scan Markdown files for broken or malformed links,
	and report errors. Exit with status code 1 if errors exist.
	"""
	parser = argparse.ArgumentParser(description="Check Markdown links in documentation.")
	parser.add_argument(
		"--grafana-hugo",
		action="store_true",
		help="Skip /docs/, /media/, /static/, /blog/, and ref: links (Grafana Hugo site).",
	)
	parser.add_argument("docs_dir", type=Path, help="Root directory of Markdown sources to scan")
	args = parser.parse_args()

	docs_dir = args.docs_dir.resolve()
	grafana_hugo = args.grafana_hugo

	if not docs_dir.exists():
		sys.exit(f"ERROR: Directory not found: {docs_dir}")

	if not docs_dir.is_dir():
		sys.exit(f"ERROR: Path is not a directory: {docs_dir}")

	files_checked = 0

	for path in docs_dir.rglob("*.md"):
		files_checked += 1
		try:
			content = path.read_text(encoding="utf-8")
		except Exception as e:
			errors.append(f"ERROR: Could not read {path}: {e}")
			continue

		src = path.relative_to(docs_dir)

		# Find all Markdown links: [text](url)
		for link in re.findall(r"\[.*?\]\((.*?)\)", content):
			link = link.strip()
			if not link:
				continue

			# Skip external links, mailto links, and internal anchors
			if link.startswith(("http", "https", "mailto:", "#")):
				continue

			# Remove anchor from link to check file existence
			link = link.split("#")[0]

			# Remove optional title from link
			link = link.split()[0]

			# Grafana documentation links shouldn't include file extension
			if link.endswith(".md"):
				error(src, link, "Link has .md extension and should end in /")
				continue

			if grafana_hugo and skip_grafana_hugo_site_link(link):
				continue

			# Strict mode (default): /docs/ must be relative — matches original links.py behavior
			if not grafana_hugo and link.startswith("/docs/"):
				error(src, link, "Link is absolute and should be relative")
				continue

			# Grafana Hugo: any other absolute path is treated as a site URL (not a repo file)
			if grafana_hugo and link.startswith("/"):
				continue

			# Determine base path for resolving relative links
			# If file is '_index.md', the base is parent directory
			# For other files, the base includes file stem
			base = path.parent if path.name == "_index.md" else path.parent / path.stem
			target = Path(os.path.normpath(base / link))

			# Existing file (e.g. .svg, .png next to the page)
			if target.is_file():
				continue

			# Check if target exists as .md file or directory with _index.md
			if target.with_suffix(".md").is_file() or (target / "_index.md").is_file():
				continue

			error(src, link, "Target does not exist")

	if files_checked == 0:
		sys.exit(f"ERROR: No Markdown files found in {docs_dir}")

	if errors:
		print("\n".join(errors))
		sys.exit(1)

	print(f"✓ Checked {files_checked} files. All links are formatted correctly and all targets exist")


if __name__ == "__main__":
	main()
