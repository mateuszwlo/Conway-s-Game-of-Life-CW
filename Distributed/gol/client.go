package gol

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"strings"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

//Sends on 8030 Listens on 8020

// Make2DArray - Makes a 2D array of dimensions specified
func Make2DArray(height, width int) [][]uint8{
	array := make([][]uint8, height)
	for i:=0; i < height; i++{
		array[i] = make([]uint8,width)
	}

	return array
}
//SavePgm - saves the pgm when broker sends SAVE event
func SavePgm(world [] [] uint8,p Params,c distributorChannels,t int){
	//Output as an image
	c.ioCommand <- ioOutput
	//format is dimension x dimension x turns
	c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(t)

	//Find alive cells and also send to image writer
	for i := 0; i < p.ImageHeight; i++{
		for j := 0; j < p.ImageWidth; j++{
			c.ioOutput <- world[i][j]
		}
	}
}
//RunGOL - runs gameoflife by calling the broker and asking it to advance turns
func RunGOL(board [][]uint8,t,width,height int,c distributorChannels,quit chan bool,p Params,servers ,listenPort, rpcPort int) [][]uint8{
	client,_:=rpc.Dial("tcp","127.0.0.1:" + strconv.Itoa(rpcPort)) //8030
	defer client.Close()
	myPort := strconv.Itoa(listenPort)
	myIP := "127.0.0.1:" + myPort //8020

	request := stubs.Request{Board: board,Turns: t,Width: width,Height: height,Addr: myIP,Servers: servers}
	response :=new(stubs.Response)

	go HandleEvent(myPort,c,quit,p) //Make Handle Update e.g. Turn Complete etc
	client.Call(stubs.AdvanceTurns,request,response)

	return response.Board
}

//HandleEvent will listen on specified port for a connection and then sends to HandleConnection
func HandleEvent(myPort string,c distributorChannels,quit chan bool,p Params){
	ln, _ := net.Listen("tcp",":"+myPort)
	go CloseListener(quit,ln)

	for {
			conn,err := ln.Accept()
			if err != nil {
				return
			}else{

				HandleConnection(conn,c,p )

			}


		}

}

//CloseListener waits on channel to close the listener
//wasn't closing normally
func CloseListener(quit chan bool,ln net.Listener)  {
	<-quit
	ln.Close()

}

// HandleConnection - Handles the incoming connection and executes the event specified from the broker
func HandleConnection(conn net.Conn,c distributorChannels,p Params){
	reader := bufio.NewReader(conn)
	msg, _ := reader.ReadString('\n')
	msgSlice := strings.Split(msg,":")
	if msgSlice[0] == "ALIVE"{
		cells, _ := strconv.Atoi(msgSlice[1])
		turns, _ := strconv.Atoi(msgSlice[2])
		c.events <- AliveCellsCount{
			CompletedTurns:turns,
			CellsCount:     cells,
		}
	}else if msgSlice[0] == "TURN"{
		turns, _ := strconv.Atoi(msgSlice[1])
		c.events <- TurnComplete{CompletedTurns:turns}

	}else if msgSlice[0] == "FLIPPED"{
		t,_ := strconv.Atoi(msgSlice[1])
		for i:=2;i< len(msgSlice)-1;i= i+ 2{

			x,_:=strconv.Atoi(msgSlice[i])
			y,_:=strconv.Atoi(msgSlice[i+1])

			c.events <- CellFlipped{CompletedTurns: t,Cell: util.Cell{X: x,Y: y}}
		}

	}else if msgSlice[0] == "SAVE"{
		t,_ := strconv.Atoi(msgSlice[1])
		world := Make2DArray(p.ImageWidth,p.ImageHeight)
		for i:=0;i<p.ImageHeight;i++{
			for j:=0;j<p.ImageWidth;j++{
				intval,_:=strconv.Atoi(msgSlice[(i*p.ImageHeight)+ j+2])
				world[i][j] = uint8(intval)
			}
		}
		go SavePgm(world,p,c,t)

	}else if msgSlice[0] == "MESSAGE"{
		fmt.Println(msgSlice[1])
	}

}



