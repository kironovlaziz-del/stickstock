from pathlib import Path

# Максимальный размер каждого итогового файла — около 800 КБ
MAX_SIZE = 800 * 1024

EXTENSIONS = {
    ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
    ".py", ".json", ".md", ".sql", ".yml", ".yaml",
    ".toml", ".css", ".scss", ".html", ".conf", ".sh",
    ".mod", ".sum", ".txt"
}

SPECIAL_FILES = {
    "Dockerfile",
    "Makefile",
    "docker-compose.yml",
    "docker-compose.yaml",
    ".dockerignore",
    ".gitignore",
    ".env.example",
    "nginx.conf",
}

EXCLUDED_DIRS = {
    ".git",
    "node_modules",
    ".next",
    "dist",
    "build",
    ".venv",
    "venv",
    "__pycache__",
    "coverage",
    ".pytest_cache",
    ".mypy_cache",
    ".idea",
    ".vscode",
}

# Эти файлы уже были загружены или могут содержать секреты
EXCLUDED_FILES = {
    ".env",
    "package-lock.json",
    "package.json",
    "tsconfig.json",
    "README.md",
    "collect_project.py",
}

files = []

for path in Path(".").rglob("*"):
    if not path.is_file():
        continue

    if any(part in EXCLUDED_DIRS for part in path.parts):
        continue

    if path.name in EXCLUDED_FILES:
        continue

    if path.name.startswith("stickstock_part_"):
        continue

    if path.suffix.lower() in EXTENSIONS or path.name in SPECIAL_FILES:
        files.append(path)

files.sort(key=lambda p: p.as_posix())

part_number = 0
current_size = 0
output = None
created_files = []
skipped_files = []


def create_output():
    global part_number, current_size, output

    if output is not None:
        output.close()

    part_number += 1
    filename = f"stickstock_part_{part_number:03d}.txt"

    output = open(filename, "w", encoding="utf-8")
    current_size = 0
    created_files.append(filename)

    print(f"Создаю: {filename}")


def write_text(text):
    global current_size

    text_size = len(text.encode("utf-8"))

    if output is None or current_size + text_size > MAX_SIZE:
        create_output()

    output.write(text)
    current_size += text_size


for path in files:
    try:
        content = path.read_text(encoding="utf-8")
    except (UnicodeDecodeError, OSError) as error:
        skipped_files.append((path.as_posix(), str(error)))
        continue

    header = (
        "\n\n"
        + "=" * 80
        + f"\nFILE: {path.as_posix()}\n"
        + "=" * 80
        + "\n\n"
    )

    full_block = header + content + "\n"
    block_size = len(full_block.encode("utf-8"))

    # Обычный случай: файл помещается целиком
    if block_size <= MAX_SIZE:
        write_text(full_block)
        continue

    # Если исходник очень большой, разделяем его по строкам
    lines = content.splitlines(keepends=True)
    chunk = ""
    chunk_number = 1

    for line in lines:
        chunk_header = (
            "\n\n"
            + "=" * 80
            + f"\nFILE: {path.as_posix()} (fragment {chunk_number})\n"
            + "=" * 80
            + "\n\n"
        )

        candidate = chunk_header + chunk + line

        if len(candidate.encode("utf-8")) > MAX_SIZE and chunk:
            write_text(chunk_header + chunk + "\n")
            chunk = line
            chunk_number += 1
        else:
            chunk += line

    if chunk:
        chunk_header = (
            "\n\n"
            + "=" * 80
            + f"\nFILE: {path.as_posix()} (fragment {chunk_number})\n"
            + "=" * 80
            + "\n\n"
        )
        write_text(chunk_header + chunk + "\n")

if output is not None:
    output.close()

print()
print(f"Обработано исходных файлов: {len(files)}")
print(f"Создано частей: {len(created_files)}")

for filename in created_files:
    size = Path(filename).stat().st_size
    print(f"  {filename}: {size / 1024:.1f} КБ")

if skipped_files:
    print()
    print("Не удалось прочитать следующие файлы:")

    for filename, reason in skipped_files:
        print(f"  {filename}: {reason}")
