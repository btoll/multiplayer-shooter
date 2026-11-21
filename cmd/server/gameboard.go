package main

import (
	"fmt"
	"maps"
	"strings"
	"sync"
)

type color int

const (
	_ color = iota
	RED
	GREEN
	YELLOW
	BLUE
	PURPLE
	_
	WHITE
)

func colorGenerator() func() (color, bool) {
	colors := []color{RED, GREEN, YELLOW, BLUE, PURPLE, WHITE}
	var i int
	return func() (color, bool) {
		if i < len(colors) {
			c := colors[i]
			i += 1
			return c, true
		}
		return -1, false
	}
}

func (c color) String() string {
	return [...]string{
		"_",
		"\033[31m",
		"\033[32m",
		"\033[33m",
		"\033[34m",
		"\033[35m",
		"_",
		"\033[37m",
	}[c]
}

type gamePieces map[int]gamepiece

type gameBoard struct {
	board     [][]int
	dimension dimension
	pieces    gamePieces
	mu        sync.RWMutex
}

type dimension struct {
	rows int
	cols int
}

func newGameboard(visibleRows, visibleCols int) *gameBoard {
	board := make([][]int, visibleRows)
	for i := range board {
		board[i] = make([]int, visibleCols)
	}
	return &gameBoard{
		board:     board,
		pieces:    make(gamePieces),
		dimension: dimension{visibleRows, visibleCols},
	}
}

func (g *gameBoard) getBoard() [][]int {
	return g.board
}

func (g *gameBoard) getDimension() dimension {
	return g.dimension
}

func (g *gameBoard) getGamepiece(id int) gamepiece {
	return g.pieces[id]
}

func (g *gameBoard) move(m msg) {
	g.mu.Lock()
	piece := g.pieces[m.id]
	if piece == nil {
		g.mu.Unlock()
		return
	}
	piece.move(m.x, m.y)
	g.mu.Unlock()
	fmt.Fprintf(mw, "[INFO] Moved piece %d %#v\n", m.id, piece.getPosition())
}

func (g *gameBoard) register(id int, kind string) {
	g.mu.Lock()
	piece := g.getGamepiece(id)
	cg := createGamepiece{}
	dimension := g.getDimension()

	if kind == "biplane" {
		cg = createGamepiece{
			Id:    int(id),
			Color: color(id),
			Kind:  "biplane",
			Position: position{
				getDivisibleBy2(dimension.rows), // All gamepieces MUST be on rows and cols that are divisible by two.
				getDivisibleBy2(dimension.cols),
			},
			Dimension: dimension,
		}
	}

	if kind == "bullet" {
		cg = createGamepiece{
			Id:        randInt(100, 1000),
			Color:     color(id),
			Kind:      "bullet",
			Position:  piece.getPosition(),
			Direction: piece.getDirection(),
			Dimension: dimension,
		}
	}

	piece = newGamePiece(cg)
	// Put the piece on the board.
	// The id is looked up when rendering and the cell colorized.
	// TODO
	switch v := piece.(type) {
	case *biplane:
		g.board[v.cur.row][v.cur.col] = cg.Id
	case *bullet:
		if v == nil {
			g.mu.Unlock()
			return
		}
		g.board[v.cur.row][v.cur.col] = cg.Id
	}
	g.pieces[cg.Id] = piece
	g.mu.Unlock()
	fmt.Fprintf(mw, "[INFO] Gamepiece created %#v\n", cg)
}

func (g *gameBoard) render() string {
	g.mu.Lock()
	rows := len(g.board)
	cols := 0
	if rows > 0 {
		cols = len(g.board[0])
	}
	cpBoard := make([][]int, rows)
	for row := range cpBoard {
		cpBoard[row] = make([]int, cols)
		copy(cpBoard[row], g.board[row])
	}
	g.mu.Unlock()

	builder := &strings.Builder{}
	builder.WriteString("\033[2J\033[H")
	for r, row := range cpBoard {
		for c, cell := range row {
			if cell == 0 {
				builder.WriteByte(' ')
			} else {
				piece := g.pieces[cell]
				// CRITICAL: Handle the case where unregister left a ghost ID
				if piece == nil {
					fmt.Println("got here", cell)
					// 1. Don't crash: treat as empty
					builder.WriteByte(' ')
					// 2. CLEAN UP: Remove the ghost ID from the REAL board
					// We must lock again to safely modify the board
					g.mu.Lock()
					// Double-check: maybe another thread cleared it already?
					if g.board[r][c] == cell {
						g.board[r][c] = 0
					}
					g.mu.Unlock()
					continue
				}
				fmt.Fprintf(builder, "%s%c\033[0m", piece.getColor(), piece.getSymbol())
			}
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}

func (g *gameBoard) unregister(id int) {
	piece := g.pieces[id]
	if piece == nil {
		return
	}
	// Clear the board slot BEFORE deleting from map
	// This ensures render() never sees a mismatch
	pos := piece.getPosition()
	g.board[pos.row][pos.col] = 0
	delete(g.pieces, id)
}

func (g *gameBoard) update() {
	g.mu.Lock()
	pieces := make(gamePieces, len(g.pieces))
	maps.Copy(pieces, g.pieces)

	var collisions []int
	for _, gamepiece := range pieces {
		if gamepiece.isStopped() {
			continue
		}
		if collisions = gamepiece.update(g); collisions != nil {
			for _, collision := range collisions {
				g.unregister(collision)
			}
		}
	}
	defer g.mu.Unlock()
	if len(collisions) > 0 {
		fmt.Fprintf(mw, "[INFO] User IDs %v unregistered.\n", collisions)
	}
}
