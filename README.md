# 🔍 File Comparison and Copy Tools

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue)](https://go.dev/)
[![Python Version](https://img.shields.io/badge/Python-3.6%2B-blue)](https://www.python.org/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/NamiLinkLabs/FileCompare/graphs/commit-activity)

This repository contains two complementary tools for comparing and copying files between directories:
1. 🚀 A Go-based file comparison tool that identifies missing files
2. 📦 A Python-based file copy tool that transfers the identified missing files

## ⚙️ Prerequisites

### For the Go Tool 🚀
- Go 1.22 or later
- Required Go packages (automatically installed via go.mod):
  - github.com/schollz/progressbar/v3
  - gopkg.in/ini.v1

### For the Python Tool 🐍
- Python 3.6 or later
- Required Python packages:
  - tqdm
  - configparser

## 🔧 Installation

1. Clone the repository:
```bash
git clone git@github.com:NamiLinkLabs/FileCompare.git
cd FIleCompare
```

2. Install Go dependencies:
```bash
go mod download
```

3. Install Python dependencies:
```bash
pip install tqdm configparser
```

## ⚡ Configuration

Create a `config.ini` file in the root directory with the following structure:

```ini
[Directories]
source_dir = /path/to/source/directory
target_dir = /path/to/target/directory

[FileTypes]
included_extensions = .jpg, .png, .pdf
excluded_extensions = .tmp

[Hashing]
large_file_threshold = 104857600
partial_hash_size = 1048576
```

## 📚 Usage

### 1. Compare Directories (Go Tool) 🔍

The Go tool scans both directories and creates a CSV file of missing files:

```bash
go run main.go
```

This will:
- Scan the source and target directories
- Create a `missing_files.csv` containing paths of files present in source but missing in target
- Generate cache files (`source_cache.json` and `target_cache.json`) to speed up future comparisons

### 2. Copy Missing Files (Python Tool) 📂

After running the comparison, use the Python tool to copy the missing files:

```bash
python copyfiles.py
```

This will:
- Read the `missing_files.csv` generated by the Go tool
- Create a `missed_files` directory in the target location
- Copy all missing files while preserving their directory structure
- Show a progress bar during the copy process

## ⚡ Performance Features

### Go Comparison Tool 🚀
- Concurrent file hashing using worker pools
- Smart hashing for large files (only hashes beginning and end)
- File hash caching to speed up subsequent runs
- Progress bar for visual feedback
- Handles permission errors gracefully

### Python Copy Tool 🐍
- Progress bar for visual feedback
- Preserves file metadata during copy
- Creates necessary directory structure automatically
- Error handling for failed copies

## 📁 Output Files

- 📄 `missing_files.csv`: List of files missing from the target directory
- 💾 `source_cache.json`: Cache of file hashes from the source directory
- 💾 `target_cache.json`: Cache of file hashes from the target directory

## ⚠️ Error Handling

Both tools include error handling for common scenarios:
- Permission denied errors
- Missing directories
- Invalid file paths
- I/O errors during copying

## 👥 Contributing

Feel free to submit issues and enhancement requests! Pull requests are welcome! 🎉

## 📝 License

MIT
