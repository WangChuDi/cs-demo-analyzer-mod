package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"google.golang.org/protobuf/proto"
)

// Grenade prices (CS2 prices)
var grenadePrices = map[common.EquipmentType]int{
	common.EqSmoke:      300,
	common.EqFlash:      200,
	common.EqHE:         300,
	common.EqIncendiary: 500,
	common.EqMolotov:    400,
	common.EqDecoy:      50,
}

// PlayerWastedUtility tracks wasted utility per player
type PlayerWastedUtility struct {
	SteamID64   uint64         `json:"steam_id_64"`
	Name        string         `json:"name"`
	TotalWasted int            `json:"total_wasted"`
	Deaths      int            `json:"deaths"`
	AvgPerDeath float64        `json:"avg_per_death"`
	Items       map[string]int `json:"items"`
}

// DemoResult stores the result from a single demo
type DemoResult struct {
	DemoFile    string                          `json:"demo_file"`
	Checksum    string                          `json:"checksum"`
	MapName     string                          `json:"map_name"`
	ServerName  string                          `json:"server_name"`
	TotalWasted int                             `json:"total_wasted"`
	TotalDeaths int                             `json:"total_deaths"`
	AvgPerDeath float64                         `json:"avg_per_death"`
	PlayerStats map[uint64]*PlayerWastedUtility `json:"player_stats"`
	Error       string                          `json:"error,omitempty"`
	ParseTimeMs int64                           `json:"parse_time_ms"`
	FromCache   bool                            `json:"from_cache"`
}

// GlobalSummary contains aggregated results
type GlobalSummary struct {
	ProcessedAt   string                          `json:"processed_at"`
	TotalDemos    int                             `json:"total_demos"`
	SuccessCount  int                             `json:"success_count"`
	FailCount     int                             `json:"fail_count"`
	CacheHits     int                             `json:"cache_hits"`
	TotalWasted   int                             `json:"total_wasted"`
	TotalDeaths   int                             `json:"total_deaths"`
	AvgPerDeath   float64                         `json:"avg_per_death"`
	DemoResults   []DemoResult                    `json:"demo_results"`
	GlobalPlayers map[uint64]*PlayerWastedUtility `json:"global_players"`
}

// Cache stores previously analyzed demos
type Cache struct {
	mu      sync.RWMutex
	Results map[string]DemoResult `json:"results"` // key = checksum
	path    string
}

func NewCache(path string) *Cache {
	c := &Cache{
		Results: make(map[string]DemoResult),
		path:    path,
	}
	c.Load()
	return c
}

func (c *Cache) Load() {
	if c.path == "" {
		return
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return // Cache file doesn't exist or can't be read
	}
	json.Unmarshal(data, &c.Results)
}

func (c *Cache) Save() error {
	if c.path == "" {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, err := json.MarshalIndent(c.Results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}

func (c *Cache) Get(checksum string) (DemoResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, ok := c.Results[checksum]
	return result, ok
}

func (c *Cache) Set(checksum string, result DemoResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Results[checksum] = result
}

var cache *Cache

func main() {
	// Command line flags
	csvInput := flag.String("csv", "", "Path to CSV file containing demo paths")
	outputJSON := flag.String("output", "", "Output JSON file path for results")
	outputCSV := flag.String("output-csv", "", "Output CSV file path for player summary")
	cacheFile := flag.String("cache", "demo_cache.json", "Cache file to avoid re-analyzing demos (set to empty to disable)")
	workers := flag.Int("workers", 0, "Number of concurrent workers (default: CPU cores)")
	verbose := flag.Bool("verbose", false, "Print detailed output for each demo")
	noCache := flag.Bool("no-cache", false, "Disable cache, analyze all demos")
	flag.Parse()

	if *workers <= 0 {
		*workers = runtime.NumCPU()
	}

	// Initialize cache
	if !*noCache && *cacheFile != "" {
		cache = NewCache(*cacheFile)
		fmt.Printf("📦 Cache loaded: %d entries\n", len(cache.Results))
	} else {
		cache = NewCache("")
	}

	// Collect demo files
	var demoFiles []string

	if *csvInput != "" {
		files, err := readDemoPathsFromCSV(*csvInput)
		if err != nil {
			log.Fatalf("Failed to read CSV: %v", err)
		}
		demoFiles = append(demoFiles, files...)
	}

	for _, arg := range flag.Args() {
		fileInfo, err := os.Stat(arg)
		if err != nil {
			log.Printf("Warning: Cannot access '%s': %v\n", arg, err)
			continue
		}

		if fileInfo.IsDir() {
			files, err := findDemoFiles(arg)
			if err != nil {
				log.Printf("Warning: Error scanning directory '%s': %v\n", arg, err)
				continue
			}
			demoFiles = append(demoFiles, files...)
		} else if strings.HasSuffix(strings.ToLower(arg), ".dem") {
			demoFiles = append(demoFiles, arg)
		}
	}

	if len(demoFiles) == 0 {
		printUsage()
		os.Exit(1)
	}

	fmt.Printf("🎮 Found %d demo file(s) to process\n", len(demoFiles))
	fmt.Printf("⚡ Using %d concurrent workers\n", *workers)
	fmt.Println("========================================")

	startTime := time.Now()

	// Process demos concurrently
	results := processDemosConcurrent(demoFiles, *workers, *verbose)

	// Save cache
	if cache.path != "" {
		if err := cache.Save(); err != nil {
			log.Printf("Warning: Failed to save cache: %v", err)
		}
	}

	// Aggregate global stats
	summary := aggregateResults(results)
	summary.ProcessedAt = time.Now().Format(time.RFC3339)

	elapsed := time.Since(startTime)

	// Print summary
	printSummary(summary, elapsed)

	// Save outputs
	if *outputJSON != "" {
		if err := saveJSON(*outputJSON, summary); err != nil {
			log.Printf("Failed to save JSON: %v", err)
		} else {
			fmt.Printf("\n📄 Results saved to: %s\n", *outputJSON)
		}
	}

	if *outputCSV != "" {
		if err := saveCSV(*outputCSV, summary); err != nil {
			log.Printf("Failed to save CSV: %v", err)
		} else {
			fmt.Printf("📊 Player summary saved to: %s\n", *outputCSV)
		}
	}
}

func printUsage() {
	fmt.Println("Usage: wasted-utility [options] [demo files or directories...]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  wasted-utility -csv demos.csv -output results.json")
	fmt.Println("  wasted-utility -csv demos.csv -cache my_cache.json")
	fmt.Println("  wasted-utility -csv demos.csv -no-cache")
	fmt.Println("  wasted-utility -workers 4 match1.dem match2.dem")
}

func readDemoPathsFromCSV(csvPath string) ([]string, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1

	var paths []string
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		path := strings.TrimSpace(record[0])
		if path == "" || strings.HasPrefix(path, "#") {
			continue
		}
		if strings.EqualFold(path, "path") || strings.EqualFold(path, "demo") || strings.EqualFold(path, "file") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(path), ".dem") {
			paths = append(paths, path)
		}
	}

	return paths, nil
}

func findDemoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".dem") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// removeInvalidUTF8 removes invalid UTF-8 sequences from a string
func removeInvalidUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue
			}
		}
		v = append(v, r)
	}
	return string(v)
}

// readVarInt32 reads a variable-length encoded int32 from reader
func readVarInt32(r io.Reader) (int32, error) {
	var result int32
	var shift uint
	for {
		var b [1]byte
		_, err := r.Read(b[:])
		if err != nil {
			return 0, err
		}
		result |= int32(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, nil
}

// calculateDemoChecksum calculates a checksum for a demo file based on its header
// This follows the same approach as cs-demo-analyzer
func calculateDemoChecksum(demoPath string) (string, string, string, error) {
	f, err := os.Open(demoPath)
	if err != nil {
		return "", "", "", err
	}
	defer f.Close()

	stats, err := f.Stat()
	if err != nil {
		return "", "", "", err
	}

	// Read filestamp (first 8 bytes)
	filestamp := make([]byte, 8)
	_, err = f.Read(filestamp)
	if err != nil {
		return "", "", "", err
	}

	isSource2 := string(filestamp) == "PBDEMS2\x00"
	if !isSource2 {
		// Fallback for non-Source 2 demos - use file-based checksum
		data := fmt.Sprintf("%s%d%d", filepath.Base(demoPath), stats.Size(), stats.ModTime().Unix())
		table := crc64.MakeTable(crc64.ECMA)
		checksum := strconv.FormatUint(crc64.Checksum([]byte(data), table), 16)
		return checksum, "", "", nil
	}

	// Source 2 demo - parse CDemoFileHeader
	// Skip 8 bytes after filestamp
	_, err = f.Read(make([]byte, 8))
	if err != nil {
		return "", "", "", err
	}

	// Read message type (should be 1 = DEM_FileHeader)
	msgType, err := readVarInt32(f)
	if err != nil {
		return "", "", "", err
	}
	if msgType != 1 {
		return "", "", "", fmt.Errorf("unexpected first proto message type: %d", msgType)
	}

	// Read tick (ignored)
	_, err = readVarInt32(f)
	if err != nil {
		return "", "", "", err
	}

	// Read message size
	size, err := readVarInt32(f)
	if err != nil {
		return "", "", "", err
	}

	// Read message bytes
	msgBytes := make([]byte, size)
	_, err = io.ReadFull(f, msgBytes)
	if err != nil {
		return "", "", "", err
	}

	// Parse CDemoFileHeader
	var header msg.CDemoFileHeader
	err = proto.Unmarshal(msgBytes, &header)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse CDemoFileHeader: %v", err)
	}

	mapName := header.GetMapName()
	serverName := removeInvalidUTF8(header.GetServerName())
	clientName := removeInvalidUTF8(header.GetClientName())

	// Build checksum data string (same as cs-demo-analyzer)
	data := fmt.Sprintf(
		"%s%s%s%d%d%s%s%d",
		mapName,
		serverName,
		clientName,
		header.GetNetworkProtocol(),
		header.GetBuildNum(),
		header.GetDemoVersionGuid(),
		header.GetDemoVersionName(),
		stats.Size(),
	)

	table := crc64.MakeTable(crc64.ECMA)
	checksum := strconv.FormatUint(crc64.Checksum([]byte(data), table), 16)

	return checksum, mapName, serverName, nil
}

func processDemosConcurrent(demoFiles []string, workers int, verbose bool) []DemoResult {
	jobs := make(chan string, len(demoFiles))
	results := make(chan DemoResult, len(demoFiles))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for demoPath := range jobs {
				// First calculate checksum
				checksum, mapName, serverName, err := calculateDemoChecksum(demoPath)
				if err != nil {
					result := DemoResult{
						DemoFile:    demoPath,
						Error:       fmt.Sprintf("checksum error: %v", err),
						PlayerStats: make(map[uint64]*PlayerWastedUtility),
					}
					results <- result
					if verbose {
						fmt.Printf("  [W%d] ❌ %s: %s\n", workerID, filepath.Base(demoPath), result.Error)
					}
					continue
				}

				// Check cache
				if cached, ok := cache.Get(checksum); ok {
					cached.DemoFile = demoPath // Update path in case it moved
					cached.FromCache = true
					results <- cached
					if verbose {
						fmt.Printf("  [W%d] 📦 %s: $%d (cached)\n", workerID, filepath.Base(demoPath), cached.TotalWasted)
					}
					continue
				}

				// Process demo
				result := processDemo(demoPath)
				result.Checksum = checksum
				result.MapName = mapName
				result.ServerName = serverName

				// Store in cache if successful
				if result.Error == "" {
					cache.Set(checksum, result)
				}

				if verbose {
					if result.Error != "" {
						fmt.Printf("  [W%d] ❌ %s: %s\n", workerID, filepath.Base(demoPath), result.Error)
					} else {
						fmt.Printf("  [W%d] ✅ %s: $%d (%dms)\n", workerID, filepath.Base(demoPath), result.TotalWasted, result.ParseTimeMs)
					}
				}
				results <- result
			}
		}(i)
	}

	for _, demoFile := range demoFiles {
		jobs <- demoFile
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []DemoResult
	processed := 0
	cacheHits := 0
	for result := range results {
		allResults = append(allResults, result)
		processed++
		if result.FromCache {
			cacheHits++
		}
		if !verbose {
			fmt.Printf("\r⏳ Processing: %d/%d demos (📦 %d cached)...", processed, len(demoFiles), cacheHits)
		}
	}
	if !verbose {
		fmt.Println()
	}

	return allResults
}

func processDemo(demoPath string) DemoResult {
	startTime := time.Now()
	result := DemoResult{
		DemoFile:    demoPath,
		PlayerStats: make(map[uint64]*PlayerWastedUtility),
	}

	f, err := os.Open(demoPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to open: %v", err)
		result.ParseTimeMs = time.Since(startTime).Milliseconds()
		return result
	}
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	p.RegisterEventHandler(func(e events.Kill) {
		victim := e.Victim
		if victim == nil {
			return
		}

		steamID := victim.SteamID64

		if _, exists := result.PlayerStats[steamID]; !exists {
			result.PlayerStats[steamID] = &PlayerWastedUtility{
				SteamID64: steamID,
				Name:      victim.Name,
				Items:     make(map[string]int),
			}
		}
		stats := result.PlayerStats[steamID]
		stats.Deaths++
		result.TotalDeaths++

		for _, weapon := range victim.Weapons() {
			if weapon == nil {
				continue
			}

			equipType := weapon.Type
			if price, isGrenade := grenadePrices[equipType]; isGrenade {
				stats.TotalWasted += price
				result.TotalWasted += price
				stats.Items[equipType.String()]++
			}
		}
	})

	err = p.ParseToEnd()
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse: %v", err)
	}

	result.ParseTimeMs = time.Since(startTime).Milliseconds()
	if result.TotalDeaths > 0 {
		result.AvgPerDeath = float64(result.TotalWasted) / float64(result.TotalDeaths)
	}

	for _, stats := range result.PlayerStats {
		if stats.Deaths > 0 {
			stats.AvgPerDeath = float64(stats.TotalWasted) / float64(stats.Deaths)
		}
	}

	return result
}

func aggregateResults(results []DemoResult) GlobalSummary {
	summary := GlobalSummary{
		DemoResults:   results,
		GlobalPlayers: make(map[uint64]*PlayerWastedUtility),
	}

	for _, result := range results {
		summary.TotalDemos++
		if result.FromCache {
			summary.CacheHits++
		}
		if result.Error != "" {
			summary.FailCount++
			continue
		}

		summary.SuccessCount++
		summary.TotalWasted += result.TotalWasted
		summary.TotalDeaths += result.TotalDeaths

		for steamID, stats := range result.PlayerStats {
			if _, exists := summary.GlobalPlayers[steamID]; !exists {
				summary.GlobalPlayers[steamID] = &PlayerWastedUtility{
					SteamID64: steamID,
					Name:      stats.Name,
					Items:     make(map[string]int),
				}
			}
			gp := summary.GlobalPlayers[steamID]
			gp.TotalWasted += stats.TotalWasted
			gp.Deaths += stats.Deaths
			for item, count := range stats.Items {
				gp.Items[item] += count
			}
		}
	}

	if summary.TotalDeaths > 0 {
		summary.AvgPerDeath = float64(summary.TotalWasted) / float64(summary.TotalDeaths)
	}
	for _, stats := range summary.GlobalPlayers {
		if stats.Deaths > 0 {
			stats.AvgPerDeath = float64(stats.TotalWasted) / float64(stats.Deaths)
		}
	}

	return summary
}

func printSummary(summary GlobalSummary, elapsed time.Duration) {
	fmt.Println("\n========================================")
	fmt.Println("     GLOBAL PLAYER SUMMARY (ALL DEMOS)")
	fmt.Println("========================================")

	type playerEntry struct {
		steamID uint64
		stats   *PlayerWastedUtility
	}
	var players []playerEntry
	for steamID, stats := range summary.GlobalPlayers {
		if stats.TotalWasted > 0 {
			players = append(players, playerEntry{steamID, stats})
		}
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].stats.TotalWasted > players[j].stats.TotalWasted
	})

	for _, p := range players {
		stats := p.stats
		fmt.Printf("\n%s (SteamID64: %d)\n", stats.Name, stats.SteamID64)
		fmt.Printf("  Total Wasted: $%d\n", stats.TotalWasted)
		fmt.Printf("  Deaths: %d\n", stats.Deaths)
		fmt.Printf("  Avg per Death: $%.2f\n", stats.AvgPerDeath)
		fmt.Printf("  Grenades lost:\n")
		for item, count := range stats.Items {
			fmt.Printf("    - %s: %d\n", item, count)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("           GRAND TOTAL")
	fmt.Println("========================================")
	fmt.Printf("Demos: %d success, %d failed, %d from cache\n", summary.SuccessCount, summary.FailCount, summary.CacheHits)
	fmt.Printf("TOTAL WASTED UTILITY: $%d\n", summary.TotalWasted)
	fmt.Printf("TOTAL DEATHS: %d\n", summary.TotalDeaths)
	fmt.Printf("AVERAGE PER DEATH: $%.2f\n", summary.AvgPerDeath)
	fmt.Printf("Processing time: %v\n", elapsed.Round(time.Millisecond))
	fmt.Println("========================================")
}

func saveJSON(path string, summary GlobalSummary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func saveCSV(path string, summary GlobalSummary) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	writer.Write([]string{
		"SteamID64", "Name", "TotalWasted", "Deaths", "AvgPerDeath",
		"Smoke", "Flash", "HE", "Incendiary", "Molotov", "Decoy",
	})

	type playerEntry struct {
		steamID uint64
		stats   *PlayerWastedUtility
	}
	var players []playerEntry
	for steamID, stats := range summary.GlobalPlayers {
		players = append(players, playerEntry{steamID, stats})
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].stats.TotalWasted > players[j].stats.TotalWasted
	})

	for _, p := range players {
		s := p.stats
		writer.Write([]string{
			fmt.Sprintf("%d", s.SteamID64),
			s.Name,
			fmt.Sprintf("%d", s.TotalWasted),
			fmt.Sprintf("%d", s.Deaths),
			fmt.Sprintf("%.2f", s.AvgPerDeath),
			fmt.Sprintf("%d", s.Items["Smoke Grenade"]),
			fmt.Sprintf("%d", s.Items["Flashbang"]),
			fmt.Sprintf("%d", s.Items["HE Grenade"]),
			fmt.Sprintf("%d", s.Items["Incendiary Grenade"]),
			fmt.Sprintf("%d", s.Items["Molotov"]),
			fmt.Sprintf("%d", s.Items["Decoy Grenade"]),
		})
	}

	return nil
}
