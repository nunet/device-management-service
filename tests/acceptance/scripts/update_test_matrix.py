import os
import re
from pathlib import Path

README_PATH = Path(__file__).parent / "../features/README.md"
FEATURE_DIR = Path(__file__).parent / "../features"
FEATURE_SECTION_HEADER = "## Feature Coverage"

def parse_feature_file(path):
    scenarios = []
    feature_name = None
    description_lines = []
    in_description = False
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            stripped = line.strip()
            if stripped.startswith("Feature:"):
                feature_name = stripped.split("Feature:", 1)[1].strip()
                in_description = True
            elif stripped.startswith("Scenario") or stripped.startswith("Scenario Outline"):
                in_description = False
                scenarios.append("- " + stripped.split(":", 1)[1].strip())
            elif in_description and stripped:
                description_lines.append(stripped)
    description = " ".join(description_lines)
    return feature_name, scenarios, description

def to_anchor(text):
    return text.lower().replace(" ", "-")

def to_filename(text):
    # Convert feature name to snake_case file name
    return text.lower().replace(" ", "_") + ".feature"

def update_feature_coverage_section(readme_lines, feature_data):
    start_index = None
    for i, line in enumerate(readme_lines):
        if line.strip() == FEATURE_SECTION_HEADER:
            start_index = i
            break
    if start_index is None:
        raise ValueError("Feature Coverage section not found.")

    new_section = [FEATURE_SECTION_HEADER, ""]

    for feature_name in sorted(feature_data.keys()):
        scenarios = sorted(feature_data[feature_name]["scenarios"])
        description = feature_data[feature_name]["description"]

        new_section.append(f"### Feature: {feature_name}\n")

        if description:
            new_section.append(f"*{description}*\n")

        if scenarios:
            new_section.append("**Scenarios:**")
            new_section.extend(scenarios)
        else:
            new_section.append("_(Scenarios to be defined)_")
        new_section.append("\n---\n")

    # Replace old section
    end_index = start_index + 1
    while end_index < len(readme_lines) and not readme_lines[end_index].startswith("## "):
        end_index += 1
    return readme_lines[:start_index] + new_section + readme_lines[end_index:]

def update_test_matrix(readme_lines, feature_data, feature_files_map):
    matrix_header = "| Feature Name |"
    table_start = next((i for i, line in enumerate(readme_lines) if line.startswith(matrix_header)), None)
    if table_start is None:
        raise ValueError("Test Matrix table not found.")

    header = readme_lines[table_start]
    separator = readme_lines[table_start + 1]

    table_end = table_start + 2
    while table_end < len(readme_lines) and readme_lines[table_end].strip().startswith("|"):
        table_end += 1

    # Keep manual data from existing rows
    existing_rows = readme_lines[table_start + 2:table_end]
    preserved_data = {}
    for row in existing_rows:
        cols = [col.strip() for col in row.strip().split("|")[1:-1]]
        if len(cols) < 7:
            continue
        match = re.search(r"\[(.*?)\]", cols[0])
        raw_name = match.group(1).strip() if match else cols[0].strip()
        preserved_data[raw_name] = cols  # All columns, including user/env/etc

    # Rebuild the updated table
    updated_table = [header, separator]
    for feature_name in sorted(feature_data.keys()):
        anchor = to_anchor(feature_name)
        file_link = feature_files_map.get(feature_name, "#")
        name_cell = f"[{feature_name}](#feature-{anchor}) ([.feature]({file_link}))"
        description = feature_data[feature_name]["description"]

        if feature_name in preserved_data:
            cols = preserved_data[feature_name]
            row = f"| {name_cell} | {description} | {' | '.join(cols[2:])} |"
        else:
            # New feature, without manual data
            row = f"| {name_cell} | {description} |  |  |  |  |  |"

        updated_table.append(row)

    return readme_lines[:table_start] + updated_table + readme_lines[table_end:]

def main():
    feature_files = list(FEATURE_DIR.glob("*.feature"))
    feature_data = {}
    feature_files_map = {}

    for path in feature_files:
        feature_name, scenarios, description = parse_feature_file(path)
        if feature_name:
            feature_data[feature_name] = {
                "scenarios": scenarios,
                "description": description
            }
            # Store relative path to the file
            feature_files_map[feature_name] = f"./{path.name}"

    with open(README_PATH, "r", encoding="utf-8") as f:
        lines = f.read().splitlines()

    lines = update_feature_coverage_section(lines, feature_data)
    lines = update_test_matrix(lines, feature_data, feature_files_map)

    with open(README_PATH, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))

    print("Test matrix updated successfully in README.md")

if __name__ == "__main__":
    main()

