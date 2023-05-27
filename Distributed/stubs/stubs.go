package stubs

//When migrating to separate project, the imports here will need changing

var AdvanceTurns ="GameOfLifeOperations.AdvanceTurns"


type Response struct {
	Board [][] uint8

}

type Request struct {
	Board [][]uint8
	Turns int
	Width int
	Height int
	Addr string
	Servers int
}


// Change Request to pass Board and Turns
// Response needs to return the list of alive cells - DONE

//Params might be needed - Image Height and Width

