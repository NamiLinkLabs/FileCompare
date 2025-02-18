import os
import shutil
import csv
import configparser
from tqdm import tqdm


def read_config(config_file):
    config = configparser.ConfigParser()
    config.read(config_file)
    return {
        'source_dir': config['Directories']['source_dir'],
        'target_dir': config['Directories']['target_dir']
    }


def copy_files(source_dir, target_dir, missing_files_csv):
    # Read the CSV file
    with open(missing_files_csv, 'r', encoding='utf-8') as f:
        reader = csv.reader(f)
        next(reader)  # Skip the header
        files_to_copy = [row[0] for row in reader]

    # Create the missed_files directory in the target directory
    missed_files_dir = os.path.join(target_dir, 'missed_files')
    os.makedirs(missed_files_dir, exist_ok=True)

    # Copy files with progress bar
    for file_path in tqdm(files_to_copy, desc="Copying files", unit="file"):
        # Get the relative path of the file
        rel_path = os.path.relpath(file_path, source_dir)

        # Create the target path
        target_path = os.path.join(missed_files_dir, rel_path)

        # Create the directory structure if it doesn't exist
        os.makedirs(os.path.dirname(target_path), exist_ok=True)

        # Copy the file
        try:
            shutil.copy2(file_path, target_path)
        except IOError as e:
            print(f"Error copying {file_path}: {e}")
        except Exception as e:
            print(f"Unexpected error copying {file_path}: {e}")


def main():
    # Read the config file
    config = read_config('config.ini')
    source_dir = config['source_dir']
    target_dir = config['target_dir']

    # CSV file path
    missing_files_csv = 'missing_files.csv'

    # Copy the files
    copy_files(source_dir, target_dir, missing_files_csv)

    print("File copying process completed.")


if __name__ == "__main__":
    main()