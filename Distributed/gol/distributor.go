package gol

import (
	"fmt"
	"net"
	"strconv"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

//CheckKey checks for a key and sends to broker on localhost
func CheckKey(port string,k <-chan rune){
	for{
		x := <- k
		conn, _ := net.Dial("tcp","127.0.0.1:"+port)
		fmt.Fprintf(conn, string(x) + "\n")
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels,k <-chan rune,servers,listenPort,rpcPort,keyPort int) {

	//2D slice to store world
	world := make([][]uint8, p.ImageWidth)
	for i:=0; i < p.ImageWidth; i++{
		world[i] = make([]uint8,p.ImageHeight)
	}

	//check keypresses
	go CheckKey(strconv.Itoa(keyPort),k) //8031

    c.ioCommand <- ioInput
	c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)

    for i := 0; i < p.ImageWidth; i++{
        for j := 0; j < p.ImageHeight; j++{
            data := <- c.ioInput

            world[i][j] = data
        }
    }

	quit := make(chan bool)

	world = RunGOL(world,p.Turns,p.ImageWidth,p.ImageHeight,c,quit,p,servers,listenPort,rpcPort)

	var aliveCells []util.Cell

	//Output as an image
	c.ioCommand <- ioOutput
	//format is dimension x dimension x turns
	c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.Turns)

	//Get alive cells and also send each cell to ioOutput for pgm writing
	for i := 0; i < p.ImageWidth; i++{
        for j := 0; j < p.ImageHeight; j++{
			c.ioOutput <- world[i][j]
			if world[i][j] == 255 {
				cell := util.Cell{X: j, Y: i}
				aliveCells = append(aliveCells, cell)
			}
        }
    }

	quit <- true //To quit listening

	c.events <- FinalTurnComplete{p.Turns, aliveCells}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}