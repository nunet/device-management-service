# Copyright 2024, Nunet
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and limitations under the License.

import os
import re

FEATURE_DIR = "../features"
EXPERIENCE_FACTOR = 1.0  # Adjusts time estimates based on team velocity
TEAM_SIZE = 2  # Number of developers

# Estimation in days per complexity
COMPLEXITY_DAYS = {
    "low": 1,
    "medium": 3,
    "high": 5
}

def find_wip_features(directory):
    feature_files = sorted([f for f in os.listdir(directory) if f.endswith(".feature")])
    total_features = 0
    total_scenarios = 0
    total_estimated_days = 0.0

    for file in feature_files:
        path = os.path.join(directory, file)
        with open(path, "r", encoding="utf-8") as f:
            lines = f.readlines()

        current_feature = None
        tags = []
        print_feature = False
        feature_lines = []
        feature_days = 0.0

        for i, line in enumerate(lines):
            stripped = line.strip()

            if stripped.startswith("@"):
                tags.extend(stripped.split())

            elif stripped.startswith("Feature:"):
                current_feature = stripped
                print_feature = "@wip" in tags
                feature_lines = []
                feature_days = 0.0
                tags = []

            elif stripped.startswith("Scenario") or stripped.startswith("Scenario Outline"):
                is_wip = print_feature or "@wip" in tags
                if is_wip:
                    complexity = "medium"

                    for tag in tags:
                        if tag.startswith("@complexity:"):
                            complexity = tag.split(":")[1].lower()

                    complexity = complexity if complexity in COMPLEXITY_DAYS else "medium"
                    base_days = COMPLEXITY_DAYS[complexity]
                    adjusted_days = base_days * EXPERIENCE_FACTOR

                    if not feature_lines:
                        feature_lines.append(f"Feature file: {file}")
                        feature_lines.append(f"  {current_feature}")
                        total_features += 1

                    feature_lines.append(f"    {stripped}")
                    total_scenarios += 1
                    feature_days += adjusted_days

                tags = []

        if feature_lines:
            print("\n".join(feature_lines))
            print(f"  Estimated days: {round(feature_days, 0):.0f}\n")
            total_estimated_days += feature_days

    print("Summary:")
    print(f"  Features with @wip: {total_features}")
    print(f"  Scenarios with @wip: {total_scenarios}")
    print(f"  Team size: {TEAM_SIZE}")
    print(f"  Total estimated days: {round(total_estimated_days / TEAM_SIZE, 0):.0f} days")

if __name__ == "__main__":
    find_wip_features(FEATURE_DIR)

