package main

import (
	"fmt"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

type GameBoard struct {
	Board *bgammon.GameState
	*cview.TextView

	dragFromX, dragFromY int8
	client               *Client

	selectionCount, selectionIndex int8
}

func NewGameBoard(client *Client) *GameBoard {
	b := &GameBoard{
		Board: &bgammon.GameState{
			Game: bgammon.NewGame(bgammon.VariantBackgammon),
		},
		TextView:       cview.NewTextView(),
		client:         client,
		selectionCount: -1,
		selectionIndex: -1,
	}
	b.TextView.SetRegions(true)
	b.TextView.SetDynamicColors(true)
	b.TextView.SetHighlightedFunc(b.handleHighlight)

	b.Update()
	return b
}

func (b *GameBoard) Update() {
	b.TextView.SetBytes(b.Board.BoardState(b.Board.PlayerNumber, true))
	if b.Board.MayDecline() && b.Board.MayOK() {
		b.TextView.Write([]byte("\n[" + colorYellow + "][\"btnresign\"][ DECLINE ][\"\"] [\"btnok\"][ ACCEPT ][\"\"][-:-:-][\"dummy\"] [\"\"]"))
	} else if b.Board.MayRoll() && b.Board.MayDouble() {
		b.TextView.Write([]byte("\n[" + colorYellow + "][\"btnroll\"][ ROLL ][\"\"] [\"btndouble\"][ DOUBLE ][\"\"][-:-:-][\"dummy\"] [\"\"]"))
	} else if b.Board.MayRoll() {
		b.TextView.Write([]byte("\n[" + colorYellow + "][\"btnroll\"][ ROLL ][\"\"][-:-:-][\"dummy\"] [\"\"]"))
	} else {
		b.TextView.Write([]byte("\n[" + colorYellow + "]"))
		if b.Board.MayReset() {
			b.TextView.Write([]byte("[\"btnreset\"][ RESET ][\"\"] "))
		}
		if b.Board.MayOK() {
			b.TextView.Write([]byte("[" + colorYellow + "][\"btnok\"][ OK ][\"\"] "))
		}
		b.TextView.Write([]byte(`[-:-:-]["dummy"] [""]`))
	}
	b.Highlight()
	app.Draw()
}

func (b *GameBoard) GetSelection() (count int8, index int8) {
	return b.selectionCount, b.selectionIndex
}

func (b *GameBoard) SetSelection(count int8, index int8) {
	if count < 0 || index < 0 {
		b.selectionCount, b.selectionIndex = -1, -1
		return
	}
	playerCheckers := bgammon.PlayerCheckers(b.Board.Board[index], b.Board.PlayerNumber)
	if count > playerCheckers {
		count = playerCheckers
	}
	b.selectionCount, b.selectionIndex = count, index
}

func (b *GameBoard) ResetSelection() {
	b.selectionCount, b.selectionIndex = -1, -1
}

func (b *GameBoard) Move(moves [][]int8) {
	if len(moves) == 0 {
		return
	}

	buf := []byte("mv ")
	for i := range moves {
		if i != 0 {
			buf = append(buf, ' ')
		}
		buf = append(buf, []byte(fmt.Sprintf("%d/%d", moves[i][0], moves[i][1]))...)
	}
	b.client.Out <- buf
	b.ResetSelection()
}

func (b *GameBoard) handleHighlight(added, removed, remaining []string) {
	defer b.Highlight()
	if len(added) > 0 {
		switch added[0] {
		case "btnroll":
			b.client.Out <- []byte("roll")
			return
		case "btnok":
			b.client.Out <- []byte("ok")
			return
		case "btndouble":
			b.client.Out <- []byte("double")
			return
		case "btnresign":
			b.client.Out <- []byte("resign")
			return
		case "btnreset":
			b.client.Out <- []byte("reset")
			return
		}
	}
}

func (b *GameBoard) mouseXYToSpace(x int8, y int8) int8 {
	// Remove padding.
	x -= 2
	y -= 1

	if (y < 1 || y == 6 || y > 11) ||
		(x < 1 || x == 19 || x == 23 || x > 43) {
		return bgammon.SpaceHomePlayer
	}

	xIndex := x - 1
	if xIndex > 18 {
		xIndex--
	}
	if xIndex > 22 {
		xIndex--
	}
	xIndex /= 3

	yIndex := y - 1
	if yIndex > 4 {
		yIndex--
	}

	return b.Board.SpaceAt(xIndex, yIndex)
}

// MouseHandler returns the mouse handler for this primitive.
func (b *GameBoard) MouseHandler() func(action cview.MouseAction, event *tcell.EventMouse, setFocus func(p cview.Primitive)) (consumed bool, capture cview.Primitive) {
	return b.WrapMouseHandler(func(action cview.MouseAction, event *tcell.EventMouse, setFocus func(p cview.Primitive)) (consumed bool, capture cview.Primitive) {
		xx, yy := event.Position()
		if !b.InRect(xx, yy) {
			return false, nil
		}
		x, y := int8(xx), int8(yy)

		switch action {
		case cview.MouseLeftDown:
			b.dragFromX, b.dragFromY = x, y
		case cview.MouseLeftUp:
			space := int8(-1)
			if b.dragFromX != x || b.dragFromY != y {
				spaceFrom := b.mouseXYToSpace(b.dragFromX, b.dragFromY)
				spaceTo := b.mouseXYToSpace(x, y)
				if spaceFrom != spaceTo {
					b.Move([][]int8{{spaceFrom, spaceTo}})
				} else {
					space = spaceFrom
				}
			} else {
				space = b.mouseXYToSpace(x, y)
			}
			if space != -1 {
				count, index := b.GetSelection()
				if count > 0 {
					if index == space {
						b.SetSelection(count+1, space)
					} else {
						move := []int8{index, space}
						moves := make([][]int8, count)
						for i := int8(0); i < count; i++ {
							moves[i] = move
						}
						b.Move(moves)
					}
				} else {
					b.SetSelection(1, space)
				}
			}
		case cview.MouseRightClick:
			b.TextView.SetHighlightedFunc(nil)
			h := b.GetHighlights()
			for i := range h {
				b.Highlight(h[i])
			}
			b.TextView.SetHighlightedFunc(b.handleHighlight)

			b.ResetSelection()

			b.Update()

			consumed = true
			return
		}

		return b.TextView.MouseHandler()(action, event, setFocus)
	})
}
