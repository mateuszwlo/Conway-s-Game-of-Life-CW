package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"
	"strings"
	"time"
	"uk.ac.bris.cs/gameoflife/brokerstubs"
	"uk.ac.bris.cs/gameoflife/stubs"
)

//Listens on 8030 and 8031 and briefly 8032 (Configurable via cmdline)
//SENDS on - 8040(worker) and 8020(client)

//PrintOnController - sends a message to the controller that should be printed
func PrintOnController(addr string,message string)  {
	conn, _ := net.Dial("tcp",addr)
	value := "MESSAGE:" + message
	fmt.Fprintf(conn,value)
	conn.Close()
}

// KeyPressesListener - Opens a connection to listen for keypresses being sent by the client
func KeyPressesListener(port string,keypress chan rune){
	ln, _ := net.Listen("tcp",":"+port)
	for{
		conn,_ := ln.Accept()
		reader := bufio.NewReader(conn)
		msg, _ := reader.ReadString('\n')
		keypress <- rune(msg[0])

	}

}

// SavePgm - Send event SAVE to the client along with world
func SavePgm(world [][]uint8,turns int,addr string,height,width int){
	conn, _ := net.Dial("tcp",addr)
	cells:= "SAVE:" + strconv.Itoa(turns)
	for i := 0; i < height; i++{
		for j := 0; j < width; j++ {
			cells = cells + ":" + strconv.Itoa(int(world[i][j]))
		}
	}

	fmt.Fprintf(conn,cells)
	conn.Close()

}

// AliveCellCountEvent - Send event ALIVE to the client along with how many turns and how many cells are alive
func AliveCellCountEvent(world [][]uint8, height, width int, r int,addr string){
	cellCount := 0

	for i := 0; i < height; i++{
		for j := 0; j < width; j++{
			if world[i][j] == 255 {
				cellCount ++
			}
		}
	}
	conn, _ := net.Dial("tcp",addr)
	value := "ALIVE:" + strconv.Itoa(cellCount) + ":" + strconv.Itoa(r)
	fmt.Fprintf(conn,value)
	conn.Close()

}

//FlippedChecker - takes in the new and old world and will return a colon separated list of cells changed
func FlippedChecker(world,newWorld [][]uint8,height,width int,addr string,turn int){
	flipped := ""
	for i:=0; i < height;i++{
		for j:=0;j < width;j++{
			if newWorld[i][j] != world[i][j]{
				flipped = flipped + ":" + strconv.Itoa(i) + ":" + strconv.Itoa(j)

			}
		}
	}
	FlippedEvent(flipped,addr,turn)
}

//FlippedEvent - Sends the FLIPPED event along with cells that have changed from FlippedChecker
func FlippedEvent(flipped, addr string,turn int){
	conn, _ := net.Dial("tcp",addr)
	//flipped turn count pairs
	value := "FLIPPED:" + strconv.Itoa(turn) + flipped
	fmt.Fprintf(conn,value)
	conn.Close()

}

//TurnCompleteEvent - Sends the TURN event - indicating that turn "turns" has been completed
func TurnCompleteEvent(turns int,addr string){
	conn, _ := net.Dial("tcp",addr)
	value := "TURN:" + strconv.Itoa(turns)
	fmt.Fprintf(conn,value)
	conn.Close()
}

//Make2DArray - Makes a 2d array of height and width
func Make2DArray(height, width int) [][]uint8{
	array := make([][]uint8, height)
	for i:=0; i < height; i++{
		array[i] = make([]uint8,width)
	}

	return array
}

//Copy2DArray - When given a 2d array will return a copy of it
// Means that functions can work byval rather than byref
func Copy2DArray(original [][]uint8) [][]uint8{
	height := len(original)
	width := len(original[0])
	array := Make2DArray(height, width)

	for i := 0; i < height; i++{
		for j := 0; j < width; j++{
			array[i][j] = original[i][j]
		}
	}
	return array
}

//Paused - Like the pause function in parallel will pause until p is pressed again
func Paused(keyPresses chan rune){
	for {
		select {
		case k := <-keyPresses:
			if k == 'p' {
				return
			}
		}
	}
}

//CallWorker - will send the worker from addr part of the world to work on via RPC
func CallWorker(StartY,EndY,width int,world [][]uint8,returnChannel chan [][]uint8,addr string){

	client,_:=rpc.Dial("tcp",addr)

	request := brokerstubs.Request{StartY: StartY,EndY:EndY,Width: width,World: world}
	response :=new(brokerstubs.Response)

	client.Call(brokerstubs.SliceAdvanceTurn,request,response)
	returnChannel<- response.Board
	client.Close()
}

//Advance - The RPC function that calculates the next turns by calling RPC on the AWS nodes
func Advance(world [][]uint8,turns,height,width int,addr string,servers int,keyPress chan rune,nodeIps []string) [][]uint8 {

	workers:= len(nodeIps)

	var workersToUse int
	//Use the smaller of the two values
	//Allows for less workers than nodes
	if workers <= servers{
		workersToUse = workers
	}else {
		workersToUse = servers
	}
	//setup ticker
	//Check which are alive and cell flipped first
	flipped := ""
	for i := 0; i < height; i++{
		for j := 0; j < width; j++{
			if world[i][j] == 255{
				flipped = flipped + ":" + strconv.Itoa(i) + ":" + strconv.Itoa(j)

			}
		}
	}
	FlippedEvent(flipped,addr,0)

	ticker := time.NewTicker(2*time.Second)

	for r := 0; r < turns; r++{

		var newWorld[][]uint8

		//check ticker
		select {
		case <- ticker.C:
			go AliveCellCountEvent(Copy2DArray(world),height, width,r,addr)
		default:
		}

		if workersToUse == 1 {
			client,_:=rpc.Dial("tcp",nodeIps[0])


			request := brokerstubs.Request{StartY: 0,EndY:height,Width: width,World: world}
			response :=new(brokerstubs.Response)

			client.Call(brokerstubs.SliceAdvanceTurn,request,response)
			newWorld = response.Board
			client.Close()
		} else{
			var channels []chan [][]uint8
			splitHeight := height / workersToUse

			for i := 0; i < workersToUse; i++{
				c := make(chan [][]uint8)
				channels = append(channels, c)

				if i == workersToUse- 1 {

					go CallWorker(splitHeight * i, height,width, world, channels[i],nodeIps[i])
				} else{

					go CallWorker(splitHeight * i, splitHeight * (i + 1), width, world, channels[i],nodeIps[i])
				}
			}

			for i := 0; i < workersToUse; i++{
				c := <- channels[i]
				newWorld = append(newWorld,c...)
				// ... to unpack slice to allow it to be appended



				}

		}

		go FlippedChecker(Copy2DArray(world),Copy2DArray(newWorld),height,width,addr,r)

		world = newWorld

		go TurnCompleteEvent(r,addr)


		//Check Keypress
		select {
		case k := <- keyPress:
			if k == 's'{
				//send req to client with world
				go SavePgm(Copy2DArray(world),r,addr,height,width)

			}else if k == 'q'{
				//r = p.Turns // Will Cause program to exit and save file at same time
				//exit the run loop so just normal return
				r = turns

			}else if k == 'p' {

				turnString := strconv.Itoa(r)
				go PrintOnController(addr,turnString)
				Paused(keyPress)
				go PrintOnController(addr,"Continuing")

			}else if k == 'k'{
				//Will also close the broker
				fmt.Println("Aws down + pgm out")

				for i := 0; i < workers ; i++ {
					client,_:=rpc.Dial("tcp",nodeIps[i])
					request := brokerstubs.Request{StartY: 0,EndY:height,Width: width,World: world}
					response :=new(brokerstubs.Response)
					client.Call(brokerstubs.Shutdown,request,response)
				}


				r = turns
			}
		default:

		}


	}
	//stop ticker
	ticker.Stop()

	return world
}

type GameOfLifeOperations struct{
	keypress chan rune
	nodeIps [] string
}
//AdvanceTurns - Runs the Advance function of the broker when called by RPC
func (s *GameOfLifeOperations) AdvanceTurns(req stubs.Request,res *stubs.Response) (err error){
	res.Board = Advance(req.Board,req.Turns,req.Height,req.Width,req.Addr,req.Servers,s.keypress,s.nodeIps)
	return
}

func main() {
	pAddr := flag.String("port","8030","Port to listen on")
	keyPressAddr := flag.String("keypress","8031","Port to listen on for keypresses")
	initAddr := flag.String("initport","8032","Port to listen on for aws nodes")
	workers := flag.String("workers","1","How many workers/servers to wait for")
	flag.Parse()


	keyPress := make(chan rune)
	//run the ports for keypress
	go KeyPressesListener(*keyPressAddr,keyPress)

	//How many aws nodes
	nodes,_ := strconv.Atoi(*workers)
	var nodeIps []string
	ln, _ := net.Listen("tcp",":"+*initAddr)
	for i:=0;i<nodes;i++{
		conn,_ := ln.Accept()
		reader := bufio.NewReader(conn)
		msg, _ := reader.ReadString('\n')
		ip := strings.Split(conn.RemoteAddr().String(),":")
		addr := ip[0] +":"+ msg
		nodeIps = append(nodeIps,addr)
		fmt.Println("Have worker")
	}
	fmt.Println("Have Required Workers")


	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GameOfLifeOperations{keyPress,nodeIps})



	listener,_ := net.Listen("tcp",":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)

}

