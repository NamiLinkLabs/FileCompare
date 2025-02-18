package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
	"gopkg.in/ini.v1"
)

type Config struct {
	SourceDir          string
	TargetDir          string
	IncludedExtensions map[string]bool
	ExcludedExtensions map[string]bool
	LargeFileThreshold int64
	PartialHashSize    int64
}

type FileHash struct {
	Path string
	Hash string
}

type HashCache struct {
	Hashes map[string]string
	mutex  sync.RWMutex
}

func loadConfig(filename string) (*Config, error) {
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{
		IncludedExtensions: make(map[string]bool),
		ExcludedExtensions: make(map[string]bool),
	}

	config.SourceDir = cfg.Section("Directories").Key("source_dir").String()
	config.TargetDir = cfg.Section("Directories").Key("target_dir").String()

	for _, ext := range cfg.Section("FileTypes").Key("included_extensions").Strings(",") {
		config.IncludedExtensions[strings.TrimSpace(ext)] = true
	}
	for _, ext := range cfg.Section("FileTypes").Key("excluded_extensions").Strings(",") {
		config.ExcludedExtensions[strings.TrimSpace(ext)] = true
	}

	config.LargeFileThreshold, _ = cfg.Section("Hashing").Key("large_file_threshold").Int64()
	config.PartialHashSize, _ = cfg.Section("Hashing").Key("partial_hash_size").Int64()

	return config, nil
}

func (cache *HashCache) Get(path string) (string, bool) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	hash, exists := cache.Hashes[path]
	return hash, exists
}

func (cache *HashCache) Set(path, hash string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.Hashes[path] = hash
}

func loadHashCache(filename string) (*HashCache, error) {
	cache := &HashCache{Hashes: make(map[string]string)}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&cache.Hashes)
	return cache, err
}

func saveHashCache(filename string, cache *HashCache) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(cache.Hashes)
}

func calculateFileHash(path string, config *Config) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	if fileInfo.Size() > config.LargeFileThreshold {
		// Hash first partial_hash_size bytes
		if _, err := io.CopyN(hash, file, config.PartialHashSize); err != nil {
			return "", err
		}
		// Move to last partial_hash_size bytes
		if _, err := file.Seek(-config.PartialHashSize, io.SeekEnd); err != nil {
			return "", err
		}
		// Hash last partial_hash_size bytes
		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
	} else {
		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func isValidPath(path string) bool {
	base := filepath.Base(path)
	return !strings.HasPrefix(base, ".") && !strings.HasPrefix(base, "$")
}

func worker(paths <-chan string, results chan<- FileHash, wg *sync.WaitGroup, config *Config, cache *HashCache, bar *progressbar.ProgressBar) {
	defer wg.Done()
	for path := range paths {
		ext := strings.ToLower(filepath.Ext(path))
		if config.IncludedExtensions[ext] && !config.ExcludedExtensions[ext] {
			if hash, exists := cache.Get(path); exists {
				results <- FileHash{Path: path, Hash: hash}
			} else {
				hash, err := calculateFileHash(path, config)
				if err != nil {

					fmt.Fprintf(os.Stderr, "Error hashing %s: %v\n", path, err)
					bar.Add(1)
					continue

				}
				cache.Set(path, hash)
				results <- FileHash{Path: path, Hash: hash}
			}
		}
		bar.Add(1)
	}
}

func getFileHashes(dir string, config *Config, cache *HashCache) (map[string][]string, error) {
	paths := make(chan string)
	results := make(chan FileHash)
	var wg sync.WaitGroup

	// Count total files
	totalFiles := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				fmt.Fprintf(os.Stderr, "Permission denied: %v\n", err)
				return filepath.SkipDir
			}
			return err
		}
		if !info.IsDir() && isValidPath(path) {
			totalFiles++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error counting files: %v", err)
	}

	// Create progress bar
	bar := progressbar.Default(int64(totalFiles))

	// Start worker goroutines
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go worker(paths, results, &wg, config, cache, bar)
	}

	// Start a goroutine to close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Start a goroutine to walk the file tree
	go func() {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsPermission(err) {
					fmt.Fprintf(os.Stderr, "Permission denied: %v\n", err)
					return filepath.SkipDir
				}
				return err
			}
			if !info.IsDir() && isValidPath(path) {
				paths <- path
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking the path %v: %v\n", dir, err)
		}
		close(paths)
	}()

	// Collect results
	fileHashes := make(map[string][]string)
	for result := range results {
		fileHashes[result.Hash] = append(fileHashes[result.Hash], result.Path)
	}

	return fileHashes, nil
}

func compareDirectories(config *Config) error {
	sourceCache, err := loadHashCache("source_cache.json")
	if err != nil {
		return err
	}
	targetCache, err := loadHashCache("target_cache.json")
	if err != nil {
		return err
	}

	fmt.Printf("Scanning source directory: %s\n", config.SourceDir)
	sourceHashes, err := getFileHashes(config.SourceDir, config, sourceCache)
	if err != nil {
		return err
	}

	fmt.Printf("\nScanning target directory: %s\n", config.TargetDir)
	targetHashes, err := getFileHashes(config.TargetDir, config, targetCache)
	if err != nil {
		return err
	}

	missingFiles := []string{}

	fmt.Println("\nProcessing missing files:")
	for hash, sourcePaths := range sourceHashes {
		if _, exists := targetHashes[hash]; !exists {
			missingFiles = append(missingFiles, sourcePaths...)
		}
	}

	// Write results to CSV
	csvFile, err := os.Create("missing_files.csv")
	if err != nil {
		return err
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	writer.Write([]string{"File Path"})
	for _, file := range missingFiles {
		writer.Write([]string{file})
	}

	fmt.Printf("\nResults saved to missing_files.csv\n")
	fmt.Printf("\nTotal files processed in source: %d\n", len(sourceHashes))
	fmt.Printf("Total files processed in target: %d\n", len(targetHashes))
	fmt.Printf("Total missing files: %d\n", len(missingFiles))

	// Save updated caches
	err = saveHashCache("source_cache.json", sourceCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving source cache: %v\n", err)
	}
	err = saveHashCache("target_cache.json", targetCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving target cache: %v\n", err)
	}

	return nil
}

func main() {
	config, err := loadConfig("config.ini")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	err = compareDirectories(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error comparing directories: %v\n", err)
		os.Exit(1)
	}
}