package gol

import (
	"fmt"
	"strconv"
	"time"
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
//Requires Dimensions on what the worker should run on and a channel to output to
func worker(startY, endY, width,imageHeight int, world func(y, x int) uint8, out chan<- [][]uint8) {
	slice := run(startY, endY, width,imageHeight, world)
	out <- slice
}

//Given Dimensions will give an empty 2d array with those dimensions
func make2DArray(height, width int) [][]uint8{
    array := make([][]uint8, height)
    for i:=0; i < height; i++{
        array[i] = make([]uint8,width)
    }

    return array
}

//Makes the world immutable by putting it in a getter function
func makeImmutableWorld(world [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return world[y][x]
	}
}
//Given the function to get the world by coods, will save the image to /out
func savePgm(world func(y, x int) uint8,p Params,c distributorChannels){
	// Tell that you want to output an image
	c.ioCommand <- ioOutput
	//format is dimension x dimension x turns
	c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.Turns)

	//Find alive cells and also send to image writer
	for i := 0; i < p.ImageHeight; i++{
		for j := 0; j < p.ImageWidth; j++{
			c.ioOutput <- world(i,j)
			}
		}
}

// paused : This will take all keypresses and will allow the other proccess to continue once p has been received
func paused(pause chan int,keyPresses <-chan rune) {
	for {
		select {
		case k := <-keyPresses:
			if k == 'p' {
				pause <- 1
				return
			}
		}
		//Could handle other keypresses like s in this state
	}
}

// Run :GOL logic - Advance a turn of the slice
func run(startY, endY, width,imageHeight int, world func(y, x int) uint8) [][]uint8{
    sliceHeight := endY - startY
    newSlice := make2DArray(sliceHeight, width)
	//Calculates size of output

        for i := 0; i < sliceHeight; i++{
            f := startY + i
            for j := 0; j < width; j++{
				//Add up the cell values of all neighbours
                sum := int(world((f + imageHeight - 1) % imageHeight,	j)) +
                    int(world((f + imageHeight - 1) % imageHeight, (j + width + 1) % width)) +
                    int(world((f + imageHeight - 1) % imageHeight, (j + width - 1) % width)) +
                    int(world(f, (j + width + 1) % width)) +
                    int(world(f, (j + width - 1) % width)) +
                    int(world((f + imageHeight + 1) % imageHeight	,j)) +
                    int(world((f + imageHeight + 1) % imageHeight	, (j + width + 1) % width)) +
                    int(world((f + imageHeight + 1) % imageHeight, (j + width - 1) % width))

                liveNeighbours := sum / 255
				// divide by 255 as alive cell value is 255 and we want to know which ones are alive

                if world(f,j) == 255{
					//If alive and
                    if liveNeighbours < 2{
						//has less than 2 neighbours
                        newSlice[i][j] = 0
						//kill it
                    } else if liveNeighbours > 3{
						// has more than 3 neighbours
                        newSlice[i][j] = 0
						//kill it
                    } else{
						//otherwise it stays alive
                        newSlice[i][j] = 255

                    }
                } else{
					//if not alive
                    if liveNeighbours == 3{
						//and has 3 neighbours
                        newSlice[i][j] = 255
						//it is now alive
                    }else{
						//otherwise still not alive
                        newSlice[i][j] = 0
                    }
                }
            }
        }
    return newSlice
}

// AliveCellsEvent :Given the function to get the world by coods,
//will send an event with the number of cells alive for the current turn
func AliveCellsEvent(world func(y, x int) uint8,c distributorChannels,p Params,turns int){
	cellCount := 0

	for i := 0; i < p.ImageHeight; i++{
		for j := 0; j < p.ImageWidth; j++{
			if world(i,j) == 255 {
				cellCount ++
			}
		}
	}
	fmt.Println(cellCount)
	c.events <- AliveCellsCount{turns,cellCount}

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels,keyPresses <-chan rune) {


	world := make2DArray(p.ImageHeight, p.ImageWidth)
	//slice to store the world.

    c.ioCommand <- ioInput
	c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
	//Say we would like to read a file

    for i := 0; i < p.ImageHeight; i++{
      for j := 0; j < p.ImageWidth; j++{
          data := <- c.ioInput
		  if data == 255{
			  c.events <- CellFlipped{0,util.Cell{X: i,Y: j}}
		  }
          world[i][j] = data
      }
    }
	//Read each cell one by one

	//setup ticker
	ticker := time.NewTicker(2*time.Second)

	var immutableWorld func(int,int) uint8

    for r := 0; r < p.Turns; r++{ //For each turn

		var newWorld [][]uint8
		immutableWorld= makeImmutableWorld(world)

		//check ticker
		select {
		case <- ticker.C:
			go AliveCellsEvent(immutableWorld,c,p,r)
			// if ticker goes off (every 2s then send alive cells event)
		default:

		}


        if p.Threads == 1 {
			//if only one thread, just run it
            newWorld = run(0, p.ImageHeight, p.ImageWidth,p.ImageHeight, immutableWorld)
        } else{
			var channels []chan [][]uint8
            height := p.ImageHeight / p.Threads
    		//if more than one, work out how much each thread should work on

            for i := 0; i < p.Threads; i++{
                c := make(chan [][]uint8)
                channels = append(channels, c)
            }
    
            for i := 0; i < p.Threads; i++{
                if i == p.Threads - 1 {
                    go worker(height * i, p.ImageHeight, p.ImageWidth,p.ImageHeight, immutableWorld, channels[i])
					//If 1 worker left assign it to the rest of the world (deals with things that don't easily divide)
                } else{
					//run worker
                    go worker(height * i, height * (i + 1), p.ImageWidth,p.ImageHeight, immutableWorld, channels[i])
                }
            }

            for i := 0; i < p.Threads; i++{
                c := <- channels[i]
				newWorld = append(newWorld,c...)
				// ... to unpack slice to allow it to be appended
            }
        }

		for i:=0;i < p.ImageHeight;i++{
			for j:=0;j<p.ImageWidth;j++{
				if newWorld[i][j] != world[i][j]{
					//send cell flipped events so SDL is updated
					c.events <- CellFlipped{CompletedTurns: r, Cell: util.Cell{X: i, Y: j}}

				}
			}
		}
		world = newWorld

		//Tell that a turn has been completed
		c.events <- TurnComplete{CompletedTurns: r}

		//Check Keypresses
		select {
		case k := <- keyPresses:
			if k == 's'{
				go savePgm(makeImmutableWorld(world),p,c)
			}else if k == 'q'{
				r = p.Turns // Will Cause program to exit and save file at same time
			}else if k == 'p' {
				fmt.Println(r)
				pause := make(chan int)
				go paused(pause,keyPresses)
				<- pause
				fmt.Println("Continuing")
				//sleeps till pause returns
			}
		default:

		}


	}
	//stop ticker
	ticker.Stop()



	var aliveCells []util.Cell

	//Output an image
	c.ioCommand <- ioOutput
	//format is dimension x dimension x turns
	c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.Turns)


	//Find alive cells and also send to image writer
	for i := 0; i < p.ImageHeight; i++{
		for j := 0; j < p.ImageWidth; j++{
			c.ioOutput <- world[i][j]
			if world[i][j] == 255 {
				cell := util.Cell{X: j, Y: i}
				aliveCells = append(aliveCells, cell)
			}
		}
	}

	//Say that we are done doing turns
	c.events <- FinalTurnComplete{p.Turns, aliveCells}

	// Make sure that the Io has finished any output before exiting.
	//fmt.Println("Before Check Idle")
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
