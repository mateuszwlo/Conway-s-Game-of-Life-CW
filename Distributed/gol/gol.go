package gol

import "flag"

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}


// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune,portsAndServers ...int) {
	//portsandServers is an optional arg - either supply all or none
	var servers,listenPort,rpcPort,keyPort int
	if len(portsAndServers) == 0{
		//set to defaults
		servers = 1
		listenPort = 8020
		rpcPort = 8030
		keyPort = 8031
	}else {
		servers = portsAndServers[0]
		listenPort = portsAndServers[1]
		rpcPort = portsAndServers[2]
		keyPort = portsAndServers[3]
	}

	flag.Parse()

	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)

    ioFilename := make(chan string)
    ioOutput := make(chan uint8)
    ioInput := make(chan uint8)

	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: ioFilename,
		output:   ioOutput,
		input:    ioInput,
	}
	go startIo(p, ioChannels)

	distributorChannels := distributorChannels{
		events:     events,
		ioCommand:  ioCommand,
		ioIdle:     ioIdle,
		ioFilename: ioFilename,
		ioOutput:   ioOutput,
		ioInput:    ioInput,
	}

	distributor(p, distributorChannels,keyPresses,servers,listenPort,rpcPort,keyPort)
	//8020 8030 8031
}