package brokerstubs



var SliceAdvanceTurn ="SliceGOLOperations.SliceAdvanceTurn"
var Shutdown ="SliceGOLOperations.Shutdown"
//Rename this to what the worker function is called

type Response struct {
	Board [][] uint8
	//slice?
}

type Request struct {
	StartY int
	EndY int
	Width int
	World [][]uint8
}


// Change Request to pass Board and Turns
// Response needs to return the list of alive cells - DONE

//Params might be needed - Image Height and Width

