#!/usr/bin/env python3
"""Strip wiresmith's per-file package-level unmarshal helpers from a .pb.go.

wiresmith emits `const maxUnmarshalDepth` and `func skipValue` into every
generated .pb.go. When two .proto files generate into the same Go package
(tempo.proto + backendwork.proto -> package tempopb) the duplicates do not
compile, so this script removes the copy from the secondary file. Run by
`make gen-proto`.
"""

import re
import subprocess
import sys

NOTE = (
    "\n// NOTE: maxUnmarshalDepth and skipValue are shared with the primary"
    "\n// .pb.go of this Go package; the duplicate copies emitted for this"
    "\n// file are stripped by `make gen-proto`"
    " (tools/strip-wiresmith-dup-helpers.py).\n"
)

BLOCK = re.compile(
    r"\nconst maxUnmarshalDepth = \d+\n\n"
    r"func skipValue\(dAtA \[\]byte, wireType int, fieldNum int32\) \(int, error\) \{"
    r".*?\n\}\n",
    re.S,
)


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: strip-wiresmith-dup-helpers.py <file.pb.go>...", file=sys.stderr)
        return 2
    for path in sys.argv[1:]:
        with open(path) as f:
            src = f.read()
        m = BLOCK.search(src)
        if not m:
            if NOTE in src:
                continue  # already stripped
            print(f"{path}: helper block not found", file=sys.stderr)
            return 1
        src = src[: m.start()] + NOTE + src[m.end() :]
        with open(path, "w") as f:
            f.write(src)
        # goimports drops imports that were only used by the stripped block.
        subprocess.run(["goimports", "-w", path], check=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
