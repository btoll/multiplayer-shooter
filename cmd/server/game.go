package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"sync"
)

var (
	logFile, _ = os.Create("debug-server.log")
	mw         = io.MultiWriter(os.Stdout, logFile)
	// TODO
	// The total number of players supported is equal to the length of the array.
	// Note that the logic needs to be written to enforce this.
	names = make(map[string]*client)
)

type game struct {
	board   gameboard
	clients map[int]*client
	mu      sync.RWMutex
}

type msg struct {
	id int
	x  int
	y  int
}

func newGame(visibleRows, visibleCols int) *game {
	return &game{
		board:   newGameboard(visibleRows, visibleCols),
		clients: make(map[int]*client),
	}
}

func randInt(min, max int) int {
	return rand.IntN(max-min+1) + min
}

func (g *game) listen(c *client) {
	go func() {
		for {
			scanner := bufio.NewScanner(c.conn)
			for scanner.Scan() {
				escapeSequence := scanner.Text()
				fmt.Fprintf(mw, "escapeSequence %q\n", escapeSequence)
				switch escapeSequence {
				case "\x1b[A":
					fallthrough
				case "k": // Up
					g.board.move(msg{
						id: c.id,
						x:  0,
						y:  -1,
					})

				case "\x1b[C":
					fallthrough
				case "l": // Right
					g.board.move(msg{
						id: c.id,
						x:  2,
						y:  0,
					})

				case "\x1b[B":
					fallthrough
				case "j": // Down
					g.board.move(msg{
						id: c.id,
						x:  0,
						y:  1,
					})

				case "\x1b[D":
					fallthrough
				case "h": // Left
					g.board.move(msg{
						id: c.id,
						x:  -2,
						y:  0,
					})

				case "\x1b": // Esc
					//					done <- struct{}{}

				case " ": // shoot
					piece := g.board.getGamepiece(c.id)
					g.board.register(createGamepiece{
						Id:        randInt(100, 1000),
						Color:     color(c.id),
						Kind:      "bullet",
						Position:  piece.getPosition(),
						Direction: piece.getDirection(),
						Dimension: g.board.(*biplanes).dimension,
					})

				case "s": // stop
					g.board.move(msg{
						id: c.id,
						x:  0,
						y:  0,
					})
				}
			}
		}
	}()
}

type createGamepiece struct {
	Id        int       `json:"id"`
	Color     color     `json:"color"`
	Kind      string    `json:"kind"`
	Dimension dimension `json:"dimension"`
	Position  position  `json:"position"`
	Direction direction `json:"direction"`
}

func (g *game) register(c *client) {
	g.clients[c.id] = c
	dimension := g.board.(*biplanes).dimension
	g.board.register(createGamepiece{
		Id:    int(c.id),
		Color: color(c.id),
		Kind:  "biplane",
		Position: position{
			getDivisibleBy2(dimension.rows), // All gamepieces MUST be on rows and cols that are divisible by two.
			getDivisibleBy2(dimension.cols),
		},
	})
	g.listen(c)
	fmt.Fprintf(mw, "[INFO] User %s registered (ID=%d).\n", c.name, c.id)
}

func (g *game) render() {
	g.mu.RLock()
	lenClients := len(g.clients)
	g.mu.RUnlock()
	if lenClients == 0 {
		return
	}
	s := g.board.render()
	var err error
	for _, client := range g.clients {
		// Very important not to send a newline to the client!
		// Doing so will enable a player to go offscreen top and bottom for many ticks.
		_, err = fmt.Fprint(client.conn, s)
		if err != nil {
			log.Printf("err=%+v\n", err)
		}
	}
}

func (g *game) update() {
	g.board.draw()
}
