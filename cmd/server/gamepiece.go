package main

import "fmt"

type gamepiece interface {
	getColor() color
	getDirection() direction
	getKind() string
	getPosition() position
	getSymbol() rune
	isStopped() bool
	move(int, int)
	setDirection(direction)
	update(*gameBoard) []int
}

type direction struct {
	x int
	y int
}

type position struct {
	row int
	col int
}

type pieceBase struct {
	id     int
	color  color
	kind   string
	symbol rune
	prev   position
	cur    position
	dir    direction
}

func (g pieceBase) getColor() color {
	return g.color
}

func (g pieceBase) getDirection() direction {
	return g.dir
}

func (g pieceBase) getKind() string {
	return g.kind
}

func (g pieceBase) getPosition() position {
	return g.cur
}

func (g pieceBase) getSymbol() rune {
	return g.symbol
}

func (g pieceBase) isStopped() bool {
	return g.dir.x == 0 && g.dir.y == 0
}

func (g *pieceBase) move(x, y int) {
	g.dir = direction{x, y}
}

func (g *pieceBase) setDirection(d direction) {
	g.dir = d
}

func newGamePiece(c createGamepiece) gamepiece {
	switch c.Kind {
	case "biplane":
		return newBiplane(c)
	case "bullet":
		return newBullet(c)
	}
	return nil
}

// Compile-time assertion, ensure that *biplane implements gamepiece without constructing
// a value (no allocation).  Checks method-set compatibililty for *biplane.
// var _ Interface = (*T)(nil)
var _ gamepiece = (*biplane)(nil)

type biplane struct {
	*pieceBase
	rules biplaneRules
}

func newBiplane(c createGamepiece) *biplane {
	return &biplane{
		pieceBase: &pieceBase{
			id:    c.Id,
			color: c.Color,
			kind:  "biplane",
			//			symbol: '🚁',
			symbol: 'X',
			cur:    c.Position,
		},
		rules: biplaneRules{},
	}
}

func (b *biplane) update(gameboard *gameBoard) []int {
	board := gameboard.getBoard()
	dim := gameboard.getDimension()
	pos := b.rules.calculateTarget(b.cur, b.dir, &dim)
	victimID, collided := b.rules.hasCollision(board, pos, b.id)
	if collided {
		collisions := []int{}
		m, n := b.rules.onCollision(b.id, victimID)
		if m {
			collisions = append(collisions, b.id)
		}
		if n {
			collisions = append(collisions, victimID)
		}
		return collisions
	}

	board[b.cur.row][b.cur.col] = 0
	b.prev.row = b.cur.row
	b.prev.col = b.cur.col
	b.cur.row = pos.row
	b.cur.col = pos.col
	board[b.cur.row][b.cur.col] = b.id
	fmt.Fprintf(mw, "[INFO] Moved piece %d %#v\n", b.id, pos)
	return nil
}

var _ gamepiece = (*bullet)(nil)

type bullet struct {
	*pieceBase
	rules bulletRules
}

func newBullet(c createGamepiece) *bullet {
	if c.Direction.x > 0 {
		c.Position.col += 4
	}
	if c.Direction.x < 0 {
		c.Position.col -= 4
	}
	if c.Direction.y > 0 {
		c.Position.row += 2
	}
	if c.Direction.y < 0 {
		c.Position.row -= 2
	}
	//	if c.position.row >= c.dimension.rows ||
	//		c.position.col >= c.dimension.cols {
	//		return nil
	//	}
	b := &bullet{
		pieceBase: &pieceBase{
			id:     c.Id,
			color:  c.Color,
			kind:   "bullet",
			symbol: '-',
			cur:    c.Position,
			dir: direction{
				x: c.Direction.x,
				y: c.Direction.y,
			},
		},
		rules: bulletRules{},
	}
	if !b.rules.boundsCheck(c.Position, c.Dimension) {
		return nil
	}
	return b
}

func increaseMagnitude(n int) int {
	if n != 0 {
		if n < 0 {
			return n - 1
		}
		return n + 1
	}
	return n
}

func (b *bullet) update(gameboard *gameBoard) []int {
	board := gameboard.getBoard()

	b.prev.row = b.cur.row
	b.prev.col = b.cur.col

	// A fired bullet should continue to outpace the biplane that fired it,
	// so increase its direction +/- 1 depending upon its direction.
	destRow := b.cur.row + increaseMagnitude(b.dir.y)
	destCol := b.cur.col + increaseMagnitude(b.dir.x)
	pos := position{destRow, destCol}

	collisions := []int{}
	if b.rules.boundsCheck(pos, gameboard.getDimension()) {
		victimID, collided := b.rules.hasCollision(board, pos, b.id)
		// if cell != 0 && cell != b.id {
		if collided {
			if gameboard.getGamepiece(b.id) == nil {
				fmt.Println("got here b.id==nil")
				return nil
			}
			if gameboard.getGamepiece(victimID) == nil {
				fmt.Println("got here victimID==nil")
				return nil
			}
			if gameboard.getGamepiece(b.id).getColor() == gameboard.getGamepiece(victimID).getColor() {
				return nil
			}
			// Remove the hit piece.
			m, n := b.rules.onCollision(b.id, victimID)
			if m {
				fmt.Println("got here1")
				collisions = append(collisions, b.id)
			}
			if n {
				fmt.Println("got here2")
				collisions = append(collisions, victimID)
			}
			return collisions
		}

		// Move the bullet.
		b.cur.row = destRow
		b.cur.col = destCol

		// No collision: Clear old position, set new position.
		board[b.prev.row][b.prev.col] = 0
		board[b.cur.row][b.cur.col] = b.id
	} else {
		// Out of bounds: Clear both old and current positions.
		board[b.prev.row][b.prev.col] = 0
		board[b.cur.row][b.cur.col] = 0
		// Unregister the bullet.
		fmt.Println("got here")
		fmt.Println("got here")
		fmt.Println("got here")
		fmt.Println("got here")
		return append(collisions, b.id)
	}
	return nil
}
