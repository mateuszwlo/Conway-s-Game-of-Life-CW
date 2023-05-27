package main

import (
	"flag"
	"fmt"
	"runtime"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/sdl"
)

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()
	var params gol.Params

	flag.IntVar(
		&params.Threads,
		"t",
		8,
		"Specify the number of worker threads to use. Defaults to 8.")

	flag.IntVar(
		&params.ImageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.ImageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")

	flag.IntVar(
		&params.Turns,
		"turns",
		10000000000,
		"Specify the number of turns to process. Defaults to 10000000000.")

	noVis := flag.Bool(
		"noVis",
		false,
		"Disables the SDL window, so there is no visualisation during the tests.")

	servers := flag.Int("servers",1,"How many servers to use - defaults to 1")
	listenPort := flag.Int("listenport",8020,"What port to listen on for events - defaults to 8020")
	rpcPort := flag.Int("rpcport",8030,"What port to RPC call the broker - defaults to 8030")
	keyPort := flag.Int("keyport",8031,"What port to send keypresses - defaults to 8031")
	flag.Parse()

	fmt.Println("Threads:", params.Threads)
	fmt.Println("Width:", params.ImageWidth)
	fmt.Println("Height:", params.ImageHeight)

	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)


	go gol.Run(params, events, keyPresses,*servers,*listenPort,*rpcPort,*keyPort)
	//8020 8030 8031
	if !(*noVis) {
		sdl.Run(params, events, keyPresses)
	} else {
		complete := false
		for !complete {
			event := <-events
			switch event.(type) {
			case gol.FinalTurnComplete:
				complete = true
			}
		}
	}
}
