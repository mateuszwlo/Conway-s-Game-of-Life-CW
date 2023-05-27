package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/brokerstubs"
)

//Listens on 8040

// Make2DArray - Makes a 2D array of dimensions specified
func Make2DArray(height, width int) [][]uint8{
	array := make([][]uint8, height)
	for i:=0; i < height; i++{
		array[i] = make([]uint8,width)
	}

	return array
}

// RunGOLSlice - runs a turn on the part of the world requested - Same logic as parallel
func RunGOLSlice(startY, endY, width int, world [][]uint8) [][]uint8{
	imageHeight := len(world)
	sliceHeight := endY - startY
	newSlice := Make2DArray(sliceHeight, width)

	for i := 0; i < sliceHeight; i++{
		f := startY + i
		for j := 0; j < width; j++{
			//Add up the cell values of all neighbours
			sum := int(world[((f + imageHeight - 1) % imageHeight)]	[j]) +
				int(world[((f + imageHeight - 1) % imageHeight)]	[((j + width + 1) % width)]) +
				int(world[((f + imageHeight - 1) % imageHeight)]	[((j + width - 1) % width)]) +
				int(world[(f)]	[((j + width + 1) % width)]) +
				int(world[(f)]	[((j + width - 1) % width)]) +
				int(world[((f + imageHeight + 1) % imageHeight)]	[j]) +
				int(world[((f + imageHeight + 1) % imageHeight)]	[((j + width + 1) % width)]) +
				int(world[((f + imageHeight + 1) % imageHeight)]	[((j + width - 1) % width)])

			liveNeighbours := sum / 255
			// divide by 255 as alive cell value is 255 and we want to know which ones are alive

			if world[f][j] == 255{
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

type SliceGOLOperations struct{}

//SliceAdvanceTurn - when called will advance the part of the world by a turn
func (s SliceGOLOperations) SliceAdvanceTurn(req brokerstubs.Request,res *brokerstubs.Response) (err error){
	res.Board = RunGOLSlice(req.StartY,req.EndY,req.Width,req.World)
	return
}
//Shutdown - when called will cleanly exit the node
func (s SliceGOLOperations) Shutdown(req brokerstubs.Request,res *brokerstubs.Response) (err error){
	os.Exit(0)
	return
}


func main()  {
	pAddr := flag.String("port","8040","Port to listen on")
	broker := flag.String("broker","127.0.0.1:8032","Ip and port to initially setup worker")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	conn, _ := net.Dial("tcp",*broker)
	fmt.Fprintf(conn,*pAddr)
	conn.Close()


	rpc.Register(&SliceGOLOperations{})
	listener,_ := net.Listen("tcp",":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}


