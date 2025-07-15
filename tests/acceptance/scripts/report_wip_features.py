import os

FEATURE_DIR = "../features"

def find_wip_features(directory):
    feature_files = sorted([f for f in os.listdir(directory) if f.endswith(".feature")])
    total_features = 0
    total_scenarios = 0

    for file in feature_files:
        path = os.path.join(directory, file)
        with open(path, "r", encoding="utf-8") as f:
            lines = f.readlines()

        current_feature = None
        tags = []
        print_feature = False
        feature_lines = []

        for line in lines:
            stripped = line.strip()

            if stripped.startswith("@"):
                tags = stripped.split()

            elif stripped.startswith("Feature:"):
                current_feature = stripped
                print_feature = "@wip" in tags
                feature_lines = []
                tags = []

            elif stripped.startswith("Scenario") or stripped.startswith("Scenario Outline"):
                is_wip = print_feature or "@wip" in tags
                if is_wip:
                    if not feature_lines:
                        feature_lines.append(f"Feature file: {file}")
                        feature_lines.append(f"  {current_feature}")
                        total_features += 1
                    feature_lines.append(f"    {stripped}")
                    total_scenarios += 1
                tags = []

        if feature_lines:
            print("\n".join(feature_lines))
            print()

    print("Summary:")
    print(f"  Features with @wip: {total_features}")
    print(f"  Scenarios with @wip: {total_scenarios}")

if __name__ == "__main__":
    find_wip_features(FEATURE_DIR)

