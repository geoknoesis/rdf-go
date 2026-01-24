package rdf

import (
	"os"
	"runtime"
	"runtime/pprof"
)

// BenchmarkProfiling provides helpers for CPU and memory profiling during benchmarks.
// Usage:
//
//	func BenchmarkMyFunction(b *testing.B) {
//		prof := NewBenchmarkProfiling("cpu.prof", "mem.prof")
//		defer prof.Close()
//		// ... benchmark code ...
//	}
type BenchmarkProfiling struct {
	cpuFile *os.File
	memFile *os.File
}

// NewBenchmarkProfiling creates profiling files for CPU and memory profiling.
// Set to empty string to disable that type of profiling.
func NewBenchmarkProfiling(cpuProfile, memProfile string) (*BenchmarkProfiling, error) {
	prof := &BenchmarkProfiling{}

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return nil, err
		}
		prof.cpuFile = f
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			return nil, err
		}
	}

	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			if prof.cpuFile != nil {
				pprof.StopCPUProfile()
				prof.cpuFile.Close()
			}
			return nil, err
		}
		prof.memFile = f
		runtime.GC() // Get accurate memory stats
	}

	return prof, nil
}

// Close stops profiling and closes files.
func (p *BenchmarkProfiling) Close() error {
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		if err := p.cpuFile.Close(); err != nil {
			return err
		}
	}

	if p.memFile != nil {
		runtime.GC() // Get accurate memory stats before writing
		if err := pprof.WriteHeapProfile(p.memFile); err != nil {
			return err
		}
		if err := p.memFile.Close(); err != nil {
			return err
		}
	}

	return nil
}
