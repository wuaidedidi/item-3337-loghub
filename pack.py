import os
import zipfile
import sys

PROJECT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_NAME = os.path.basename(PROJECT_DIR)
OUTPUT_DIR = os.path.join(os.path.dirname(PROJECT_DIR), "交付产物")

EXCLUDE_DIRS = {
    "node_modules",
    ".git",
    ".windsurf",
    ".omc",
    ".agent",
    ".playwright-mcp",
    "docs",
    "venv",
    "__pycache__",
    ".next",
    "dist",
    ".nuxt",
}

EXCLUDE_FILES = {
    "result.md",
    "rule.md",
    "Rule.md",
    "CLAUDE.md",
    "轨迹.md",
    "项目轨迹.md",
    "pack.py",
    "nul",
    ".gitignore",
}

EXCLUDE_EXTENSIONS = {
    ".pyc",
    ".pyo",
    ".log",
}


def should_exclude(rel_path, filename):
    parts = rel_path.replace("\\", "/").split("/")
    for part in parts:
        if part in EXCLUDE_DIRS:
            return True
    if filename in EXCLUDE_FILES:
        return True
    _, ext = os.path.splitext(filename)
    if ext in EXCLUDE_EXTENSIONS:
        return True
    # exclude empty testproducer dir
    if "testproducer" in parts and filename == "":
        return True
    return False


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    zip_path = os.path.join(OUTPUT_DIR, f"{PROJECT_NAME}.zip")

    file_count = 0
    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for root, dirs, files in os.walk(PROJECT_DIR):
            # filter out excluded directories in-place
            dirs[:] = [d for d in dirs if d not in EXCLUDE_DIRS]

            for f in files:
                full_path = os.path.join(root, f)
                rel_path = os.path.relpath(full_path, os.path.dirname(PROJECT_DIR))

                if should_exclude(os.path.relpath(root, PROJECT_DIR), f):
                    continue

                zf.write(full_path, rel_path)
                file_count += 1

    size_mb = os.path.getsize(zip_path) / (1024 * 1024)
    print(f"Done! {file_count} files packed.")
    print(f"Output: {zip_path}")
    print(f"Size: {size_mb:.2f} MB")


if __name__ == "__main__":
    main()
