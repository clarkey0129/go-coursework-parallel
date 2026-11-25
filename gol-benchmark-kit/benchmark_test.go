package main

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

const benchLength = 1000

func BenchmarkGol(b *testing.B) {
	for _, size := range []int{64, 128, 256, 512} { //several board sizes
		for threads := 1; threads <= 16; threads++ {
			os.Stdout = nil // Disable all program output apart from benchmark results
			p := gol.Params{
				Turns:       benchLength,
				Threads:     threads,
				ImageWidth:  size,
				ImageHeight: size,
			}
			name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
			b.Run(name, func(b *testing.B) {
				var before, after runtime.MemStats // counts garbage collections for the memory strategy comparison
				runtime.ReadMemStats(&before)
				for i := 0; i < b.N; i++ {
					events := make(chan gol.Event)
					go gol.Run(p, events, nil)
					for range events {

					}
				}
				runtime.ReadMemStats(&after)
				b.ReportMetric(float64(after.NumGC-before.NumGC)/float64(b.N), "gc-cycles/op")               //number of gc cycles per run
				b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/1e6/float64(b.N), "gc-ms/op") //total gc pause per run in ms
			})
		}
	}
}
