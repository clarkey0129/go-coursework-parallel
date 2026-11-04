package gol

import (
	"fmt"
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
	keyPresses <-chan rune
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	world := loadWorld(p, c)                       //loads world
	next := makeWorld(p.ImageHeight, p.ImageWidth) //creates world to write next state to

	turn := 0
	c.events <- StateChange{turn, Executing}

	initial := calculateAliveCells(p, world) //calculates intial number of cells alive
	if len(initial) > 0 {
		c.events <- CellsFlipped{CompletedTurns: turn, Cells: initial}
	}
	aliveCount := len(initial)

	jobs := make([]chan job, p.Threads)   //creates slots for channels according to the number of threads
	out := make([]chan result, p.Threads) //creates slots for results channels
	for i := 0; i < p.Threads; i++ {
		jobs[i] = make(chan job)   //creates jobs channel
		out[i] = make(chan result) //creates results channel
		startY := i * p.ImageHeight / p.Threads
		endY := (i + 1) * p.ImageHeight / p.Threads                  //sets rows for worker
		go worker(startY, endY, 0, p.ImageWidth, p, jobs[i], out[i]) //assigns worker its strip and channels
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	quit := false
	// TODO: Execute all turns of the Game of Life.

	for turn < p.Turns && !quit {

		select {
		case <-ticker.C: //sends AliceCellsCount when appropriate
			c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: aliveCount}
		case key := <-c.keyPresses:
			quit = handleKey(key, p, c, world, turn) //handles key presses
		default: //by default doesn't stop
		}
		if quit { //if 'q' is pressed, handleKey sets quit to true above and the program stops
			break
		}

		for _, j := range jobs {
			j <- job{world: world, newWorld: next} //goes through each turn (one iteration of the for loop
		} //sends the given worker its next turn and the next iteration
		total := 0 //immediately sends the next worker its task
		for i := range out {
			r := <-out[i]    //same process but sends results
			total += r.alive //counts alive cells strip by strip
		}

		world, next = next, world //swaps worlds so there's no need to create a fresh world each time
		turn++
		aliveCount = total
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	for _, j := range jobs {
		close(j) //closes job channels
	}

	saveWorld(p, c, world, turn)

	c.events <- StateChange{turn, Quitting}

	close(c.events)
}

const (
	alive = 255
	dead  = 0
)

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height) //creates a slice of slices/2d slice
	for y := range world {          //goes through each row
		world[y] = make([]byte, width) //sets row value to a slice of bytes (width) long
	}
	return world
}

func loadWorld(p Params, c distributorChannels) [][]byte {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	world := makeWorld(p.ImageHeight, p.ImageWidth)
	for y := range world {
		for x := range world[y] {
			world[y][x] = <-c.ioInput //inputs loads world row by row
		}
	}
	return world
}

func saveWorld(p Params, c distributorChannels, world [][]byte, turn int) { //mirrors load world
	filename := fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn) //sets filename

	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for y := range world {
		for x := range world[y] {
			c.ioOutput <- world[y][x] //sends through world row by row
		}
	}
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
}

type job struct{ world, newWorld [][]byte } // creates a job for a worker (passed the current board and a board to write to)

type result struct {
	flipped []util.Cell // cells in this strip that changed state this turn
	alive   int         // live cells in this strip after this turn
} // result is what a worker hands back to the distributor after a turn.

func worker(startY, endY, startX, endX int, p Params, jobs <-chan job, out chan<- result) {
	for j := range jobs { //keeps channel open so keeps receiving & processing values until the channel is closed
		out <- calculateNextState(p, startY, endY, startX, endX, j.world, j.newWorld) //calculates new state of assigned rows
	}
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var cells []util.Cell
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == alive {
				cells = append(cells, util.Cell{X: x, Y: y})
			}
		}
	}
	return cells
}

func calculateNextState(p Params, startY, endY, startX, endX int, world, newWorld [][]byte) result {
	var res result
	height, width := p.ImageHeight, p.ImageWidth

	for y := startY; y < endY; y++ {

		//works out which rows are above, below and on the level of y
		u := y - 1
		if u < 0 {
			u = height - 1 //accounting for wrapping
		}
		d := y + 1
		if d == height {
			d = 0 //wrapping again
		}
		up := world[u]
		mid := world[y]
		down := world[d]
		out := newWorld[y] //needs to be copied into new world not changed as it goes

		for x := startX; x < endX; x++ {
			l := x - 1
			if l < 0 {
				l = width - 1 // wrapping left
			}
			r := x + 1
			if r == width {
				r = 0 // wrapping right
			}

			n := up[l]&1 + up[x]&1 + up[r]&1 +
				mid[l]&1 + mid[r]&1 +
				down[l]&1 + down[x]&1 + down[r]&1 //counts number of surrounding live cells

			old := mid[x]
			next := byte(dead)
			if n == 3 || (n == 2 && old == alive) {
				next = alive
				res.alive++
			}
			out[x] = next

			if next != old {
				res.flipped = append(res.flipped, util.Cell{X: x, Y: y})
			}
		}
	}

	return res
}

func handleKey(key rune, p Params, c distributorChannels, world [][]byte, turn int) bool {
	switch key {
	case 's':
		saveWorld(p, c, world, turn)
	case 'q':
		return true //quits
	case 'p':
		c.events <- StateChange{turn, Paused}
		fmt.Printf("Paused on turn %d\n", turn)
		// Block until resumed, 's' and 'q' still work while paused.
		for key := range c.keyPresses {
			switch key {
			case 's':
				saveWorld(p, c, world, turn)
			case 'q':
				return true
			case 'p':
				c.events <- StateChange{turn, Executing}
				fmt.Println("Continuing")
				return false //continues
			}
		}
	}
	return false
}
