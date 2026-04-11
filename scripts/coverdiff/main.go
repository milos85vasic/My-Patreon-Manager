package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// parse reads `go tool cover -func` output and returns per-package average coverage
// and the reported total.
func parse(r io.Reader) (map[string]float64, float64, error) {
	pkgSum := map[string]float64{}
	pkgCount := map[string]int{}
	var total float64
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return nil, 0, fmt.Errorf("malformed total line: %q", line)
			}
			pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
			v, err := strconv.ParseFloat(pctStr, 64)
			if err != nil {
				return nil, 0, err
			}
			total = v
			continue
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		filePath := line[:colon]
		pkg := path.Dir(filePath)
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		v, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			continue
		}
		pkgSum[pkg] += v
		pkgCount[pkg]++
	}
	if err := s.Err(); err != nil {
		return nil, 0, err
	}
	out := map[string]float64{}
	for k, sum := range pkgSum {
		out[k] = sum / float64(pkgCount[k])
	}
	return out, total, nil
}

func enforce(pkgs map[string]float64, totalPct, minPct float64) error {
	var bad []string
	for k, v := range pkgs {
		if v+1e-9 < minPct {
			bad = append(bad, fmt.Sprintf("%s: %.2f%%", k, v))
		}
	}
	sort.Strings(bad)
	if totalPct+1e-9 < minPct {
		bad = append(bad, fmt.Sprintf("TOTAL: %.2f%%", totalPct))
	}
	if len(bad) > 0 {
		return errors.New("packages below threshold:\n  " + strings.Join(bad, "\n  "))
	}
	return nil
}

func main() {
	minPct := flag.Float64("min", 100.0, "minimum per-package and total coverage percent")
	flag.Parse()
	pkgs, total, err := parse(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error:", err)
		os.Exit(2)
	}
	if err := enforce(pkgs, total, *minPct); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("OK total=%.2f%% across %d packages (min=%.1f%%)\n", total, len(pkgs), *minPct)
}
