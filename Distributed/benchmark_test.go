package main
import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)
const benchLength = 100
const listenPort = 8020
const rpcPort = 8030
const keyPort = 8031
func BenchmarkGol(b *testing.B) {
	for threads := 1; threads <= 8; threads++ {
		os.Stdout = nil // Disable all program output apart from benchmark results
		p := gol.Params{
			Turns:       benchLength,
			Threads:     threads,
			ImageWidth:  512,
			ImageHeight: 512,
		}
		name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				events := make(chan gol.Event)
				go gol.Run(p, events, nil,threads,listenPort,rpcPort,keyPort)
				for range events {
				}
			}
		})
	}
}
