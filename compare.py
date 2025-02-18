import subprocess
import json
import os
import csv
from typing import Dict, Optional, Tuple

class CompareTools:
    def __init__(self, config_path: str = "config.ini"):
        """Initialize the comparison tools with a config file path."""
        self.config_path = config_path
        self._validate_go_installation()

    def _validate_go_installation(self) -> None:
        """Validate that Go is installed and accessible."""
        try:
            subprocess.run(["go", "version"], check=True, capture_output=True)
        except subprocess.CalledProcessError:
            raise RuntimeError("Go is not installed or not accessible")
        except FileNotFoundError:
            raise RuntimeError("Go executable not found in PATH")

    def compare_directories(self) -> Tuple[Dict[str, str], int]:
        """
        Compare directories using the Go tool.
        
        Returns:
            Tuple containing:
            - Dictionary of missing files
            - Total number of missing files
        """
        try:
            # Run the Go program with config path
            result = subprocess.run(
                ["go", "run", "main.go", self.config_path],
                check=True,
                capture_output=True,
                text=True
            )

            # Read the results from missing_files.csv
            missing_files = {}
            if os.path.exists("missing_files.csv"):
                with open("missing_files.csv", "r") as f:
                    # Skip header
                    next(f)
                    for line in f:
                        path = line.strip()
                        if path:
                            missing_files[path] = os.path.basename(path)

            # Get cache information
            source_cache = self._read_cache("source_cache.json")
            target_cache = self._read_cache("target_cache.json")

            return missing_files, len(missing_files)

        except subprocess.CalledProcessError as e:
            raise RuntimeError(f"Error running Go comparison tool: {e.stderr}")

    def _read_cache(self, cache_file: str) -> Optional[Dict[str, str]]:
        """Read a cache file and return its contents."""
        try:
            if os.path.exists(cache_file):
                with open(cache_file, "r") as f:
                    return json.load(f)
        except json.JSONDecodeError:
            print(f"Warning: Could not decode {cache_file}")
        except Exception as e:
            print(f"Warning: Error reading {cache_file}: {e}")
        return None

    def get_cache_stats(self) -> Dict[str, int]:
        """Get statistics about the cache files."""
        stats = {}
        for cache_file in ["source_cache.json", "target_cache.json"]:
            cache = self._read_cache(cache_file)
            stats[cache_file] = len(cache) if cache else 0
        return stats

    def get_duplicates(self) -> Tuple[Dict[str, list], Dict[str, list]]:
        """
        Get duplicate files from source and target directories.
        
        Returns:
            Tuple containing:
            - Dictionary of source duplicates (hash -> list of paths)
            - Dictionary of target duplicates (hash -> list of paths)
        """
        source_dups = {}
        target_dups = {}
        
        # Read duplicate files from CSVs
        for csv_file, dups_dict in [("source_dups.csv", source_dups), 
                                  ("target_dups.csv", target_dups)]:
            if os.path.exists(csv_file):
                with open(csv_file, 'r') as f:
                    reader = csv.reader(f)
                    next(reader)  # Skip header
                    for row in reader:
                        if len(row) >= 2:
                            hash_val, paths = row
                            dups_dict[hash_val] = paths.split("|")
        
        return source_dups, target_dups

def main():
    """Example usage of the CompareTools class."""
    compare = CompareTools()
    
    try:
        missing_files, total_missing = compare.compare_directories()
        print(f"\nFound {total_missing} missing files")
        
        if total_missing > 0:
            print("\nFirst 5 missing files:")
            for i, (path, name) in enumerate(list(missing_files.items())[:5]):
                print(f"{i+1}. {path}")

        # Get and display duplicates
        source_dups, target_dups = compare.get_duplicates()
        
        print(f"\nFound {len(source_dups)} duplicate sets in source directory")
        if source_dups:
            print("\nFirst 3 source duplicates:")
            for i, (hash_val, paths) in enumerate(list(source_dups.items())[:3]):
                print(f"{i+1}. Hash: {hash_val[:8]}...")
                for path in paths[:2]:  # Show only first 2 paths
                    print(f"   - {path}")
                if len(paths) > 2:
                    print(f"   - ... and {len(paths)-2} more")

        print(f"\nFound {len(target_dups)} duplicate sets in target directory")
        if target_dups:
            print("\nFirst 3 target duplicates:")
            for i, (hash_val, paths) in enumerate(list(target_dups.items())[:3]):
                print(f"{i+1}. Hash: {hash_val[:8]}...")
                for path in paths[:2]:  # Show only first 2 paths
                    print(f"   - {path}")
                if len(paths) > 2:
                    print(f"   - ... and {len(paths)-2} more")

        cache_stats = compare.get_cache_stats()
        print("\nCache statistics:")
        for cache_file, count in cache_stats.items():
            print(f"{cache_file}: {count} entries")

    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
