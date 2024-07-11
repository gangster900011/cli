package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

const (
	ScreenLobby = iota
	ScreenGame
)

var colorYellow = "#FFFF00"

var (
	app            *cview.Application
	uiGrid         *cview.Grid
	gameList       *cview.List
	gameListFooter *cview.TextView
	gameListGrid   *cview.Grid
	confirmExit    *cview.TextView
	board          *GameBoard
	boardGrid      *cview.Grid
	gameBuffer     *cview.TextView
	statusBuffer   *cview.TextView
	inputField     *cview.InputField
	loginForm      *cview.Form

	createGameForm          *cview.Form
	createGamePointsField   *cview.InputField
	createGamePasswordField *cview.InputField
	createGameGrid          *cview.Grid

	joinGameForm          *cview.Form
	joinGameLabelField    *cview.TextView
	joinGamePasswordField *cview.InputField
	joinGameGrid          *cview.Grid

	statusWriter *bufferWriter
	gameWriter   *bufferWriter

	viewScreen  int
	inputMode   bool
	screenWidth int

	showJoinGameDialog bool
	joinGameID         int
	joinGameName       string

	allGames             []bgammon.GameListing
	autoRefresh          = true
	showCreateGameDialog bool
	gameInProgress       bool

	spacerBox = cview.NewBox()
)

const loginFooterText = `To log in as a guest, enter a username (if you want) and
  click connect, without entering a password.

For information on how to play backgammon, visit:
  https://bkgm.com/rules.html

For information on bgammon-cli (this client), visit:
  https://code.rocket9labs.com/tslocum/bgammon-cli`

type bufferWriter struct {
	Buffer *cview.TextView
}

func (w *bufferWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	app.Draw(w.Buffer)
	return n, err
}

func logIn(c *Client) {
	if c.connecting {
		return
	}
	c.connecting = true

	app.SetRoot(uiGrid, true)
	updateFocus()
	hideCursor()

	l("*** Connecting...")
	go c.Connect()
}

func setScreen(screen int) {
	viewScreen = screen
	buildLayout()
}

func primitiveInForm(p cview.Primitive, form *cview.Form) bool {
	if p == form {
		return true
	}
	for i := 0; i < form.GetFormItemCount(); i++ {
		if p == form.GetFormItem(i) {
			return true
		}
	}
	for i := 0; i < form.GetButtonCount(); i++ {
		if p == form.GetButton(i) {
			return true
		}
	}
	return false
}

func beforeFocus(p cview.Primitive) bool {
	if !board.client.LoggedIn() {
		return primitiveInForm(p, loginForm)
	}

	if inputMode {
		return p == inputField
	}

	if viewScreen == ScreenLobby {
		if showCreateGameDialog || showJoinGameDialog {
			return p != gameList && p != gameBuffer && p != statusBuffer && p != inputField
		} else if gameInProgress {
			return p == inputField
		} else {
			return p == gameList
		}
	} else {
		return p == inputField
	}
}

func hideCursor() {
	screen := app.GetScreen()
	if screen != nil {
		screen.HideCursor()
	}
}

func updateFocus() {
	if viewScreen == ScreenLobby {
		if inputMode {
			app.SetFocus(inputField)
		} else if showCreateGameDialog {
			createGameForm.SetFocus(0)
			app.SetFocus(createGameForm)
		} else if showJoinGameDialog {
			joinGameForm.SetFocus(0)
			app.SetFocus(joinGameForm)
		} else if gameInProgress {
			app.SetFocus(inputField)
		} else {
			app.SetFocus(gameList)
		}
	} else {
		if inputMode {
			app.SetFocus(inputField)
		} else {
			app.SetFocus(gameBuffer)
			hideCursor()
		}
	}
}

func buildLayout() {
	autoRefreshStatus := "enabled"
	if !autoRefresh {
		autoRefreshStatus = "disabled"
	}
	gameListFooter.SetText("[" + colorYellow + "][\"btncreate\"][ Create match ][\"\"]    [\"btnrefresh\"][ Refresh ][\"\"][-:-:-][\"dummy\"]    [" + colorYellow + "][\"btnautorefresh\"][ Auto-refresh " + autoRefreshStatus + " ][\"\"][-:-:-][\"dummy\"] [\"\"]")

	uiGrid.Clear()

	var currentScreen cview.Primitive
	if viewScreen == ScreenLobby {
		if showCreateGameDialog {
			currentScreen = createGameGrid
		} else if showJoinGameDialog {
			currentScreen = joinGameGrid
		} else if gameInProgress {
			currentScreen = confirmExit
		} else {
			currentScreen = gameListGrid
		}
	} else {
		currentScreen = boardGrid
	}
	updateFocus()

	var bottomField cview.Primitive
	if inputMode {
		bottomField = inputField
	} else {
		bottomField = spacerBox
	}

	const boardHeight = 16

	// Single column on smaller screens
	const dualColumnWidth = 156
	if screenWidth < dualColumnWidth {
		uiGrid.SetRows(boardHeight, 1, -1, 1, -1, 1)
		uiGrid.SetColumns(-1)
		uiGrid.AddItem(currentScreen, 0, 0, 1, 1, 0, 0, false)
		uiGrid.AddItem(cview.NewBox(), 1, 0, 1, 1, 0, 0, false)
		uiGrid.AddItem(gameBuffer, 2, 0, 1, 1, 0, 0, false)
		uiGrid.AddItem(cview.NewBox(), 3, 0, 1, 1, 0, 0, false)
		uiGrid.AddItem(statusBuffer, 4, 0, 1, 1, 0, 0, false)
		uiGrid.AddItem(bottomField, 5, 0, 1, 1, 0, 0, true)
		return
	}

	uiGrid.SetRows(boardHeight, 1, -1, 1)
	uiGrid.SetColumns(-1, 1, -1)
	uiGrid.AddItem(currentScreen, 0, 0, 1, 3, 0, 0, false)
	uiGrid.AddItem(cview.NewBox(), 1, 0, 1, 3, 0, 0, false)
	uiGrid.AddItem(gameBuffer, 2, 0, 1, 1, 0, 0, false)
	uiGrid.AddItem(cview.NewBox(), 2, 1, 1, 1, 0, 0, false)
	uiGrid.AddItem(statusBuffer, 2, 2, 1, 1, 0, 0, false)
	uiGrid.AddItem(bottomField, 3, 0, 1, 3, 0, 0, true)
}

func resetCreateGameDialog() {
	nameInput := createGameForm.GetFormItem(0).(*cview.InputField)
	pointsInput := createGameForm.GetFormItem(1).(*cview.InputField)
	typeDropDown := createGameForm.GetFormItem(2).(*cview.DropDown)
	passwordInput := createGameForm.GetFormItem(3).(*cview.InputField)

	nameInput.SetText("")
	pointsInput.SetText("1")
	typeDropDown.SetCurrentOption(0)
	passwordInput.SetText("")
	passwordInput.SetVisible(false)
}

func acceptCreateGameDialog() {
	if viewScreen == ScreenGame {
		return
	}

	nameInput := createGameForm.GetFormItem(0).(*cview.InputField)
	pointsInput := createGameForm.GetFormItem(1).(*cview.InputField)
	typeDropDown := createGameForm.GetFormItem(2).(*cview.DropDown)
	passwordInput := createGameForm.GetFormItem(3).(*cview.InputField)

	typeAndPassword := "public"
	index, _ := typeDropDown.GetCurrentOption()
	if index == 1 {
		if strings.TrimSpace(passwordInput.GetText()) == "" {
			createGameForm.SetFocus(3)
			app.SetFocus(createGameForm)
			l("Please enter a password to create a private match.")
			return
		}
		typeAndPassword = fmt.Sprintf("private %s", strings.ReplaceAll(passwordInput.GetText(), " ", "_"))
	}

	points, err := strconv.Atoi(pointsInput.GetText())
	if err != nil {
		points = 1
	}

	board.client.Out <- []byte(fmt.Sprintf("c %s %d %s", typeAndPassword, points, nameInput.GetText()))
}

func RunApp(c *Client, b *GameBoard) error {
	app.EnableMouse(true)

	app.SetBeforeFocusFunc(beforeFocus)

	gameBuffer = cview.NewTextView()
	gameBuffer.SetVerticalAlign(cview.AlignBottom)
	gameBuffer.SetScrollable(true)
	gameBuffer.SetScrollBarVisibility(cview.ScrollBarAlways)

	statusBuffer = cview.NewTextView()
	statusBuffer.SetVerticalAlign(cview.AlignBottom)
	statusBuffer.SetScrollable(true)
	statusBuffer.SetScrollBarVisibility(cview.ScrollBarAlways)

	confirmExit = cview.NewTextView()
	confirmExit.SetText("Are you sure you want to leave the match?\n\n(Press Y/N)")
	confirmExit.SetTextAlign(cview.AlignCenter)
	confirmExit.SetVerticalAlign(cview.AlignMiddle)

	inputField = cview.NewInputField()
	inputField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if inputMode {
			return true
		}

		if viewScreen == ScreenLobby {
			if showCreateGameDialog {
				acceptCreateGameDialog()
			} else if gameInProgress {
				if lastChar == 'y' || lastChar == 'Y' {
					board.client.Out <- []byte("leave")
				} else if lastChar == 'n' || lastChar == 'N' {
					setScreen(ScreenGame)
				}
			}
			return false
		} else if !inputMode {
			return false
		}
		return true
	})
	inputField.SetFieldBackgroundColor(cview.Styles.PrimitiveBackgroundColor)
	inputField.SetFieldBackgroundColorFocused(cview.Styles.PrimitiveBackgroundColor)

	boardGrid = cview.NewGrid()
	boardGrid.SetColumns(2, -1)
	boardGrid.SetRows(1, 15)
	boardGrid.AddItem(cview.NewBox(), 0, 0, 1, 2, 0, 0, false)
	boardGrid.AddItem(cview.NewBox(), 1, 0, 1, 1, 0, 0, false)
	boardGrid.AddItem(board, 1, 1, 1, 1, 0, 0, true)

	loginForm = cview.NewForm()
	usernameField := cview.NewInputField()
	usernameField.SetLabel("Username")
	usernameField.SetFieldWidth(16)
	usernameField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		return !unicode.IsSpace(lastChar)
	})
	loginPasswordField := cview.NewInputField()
	loginPasswordField.SetLabel("Password")
	loginPasswordField.SetFieldWidth(16)
	loginPasswordField.SetMaskCharacter('*')
	loginForm.AddFormItem(usernameField)
	loginForm.AddFormItem(loginPasswordField)

	connectFunc := func() {
		c.Username = usernameField.GetText()
		c.Password = loginPasswordField.GetText()

		logIn(c)
	}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		handleCreateGameInput := viewScreen == ScreenLobby && showCreateGameDialog
		handleJoinGameInput := viewScreen == ScreenLobby && showJoinGameDialog
		handleGameInput := viewScreen == ScreenGame && !inputMode && gameInProgress

		switch event.Key() {
		case tcell.KeyEnter:
			if !c.LoggedIn() {
				connectFunc()
				return nil
			} else if inputMode {
				text := inputField.GetText()
				if len(text) == 0 {
					inputMode = false
					buildLayout()
					return nil
				}

				if text[0] == '/' {
					text = text[1:]
				} else {
					l(fmt.Sprintf("<%s> %s", c.Username, text))
					text = "say " + text
				}

				c.Out <- []byte(text)
				inputField.SetText("")
				return nil
			} else if handleCreateGameInput {
				if app.GetFocus() == createGameForm.GetButton(0) {
					showCreateGameDialog = false
					buildLayout()
				} else {
					acceptCreateGameDialog()
				}
				return nil
			} else if handleJoinGameInput {
				if app.GetFocus() == joinGameForm.GetButton(0) {
					showJoinGameDialog = false
					buildLayout()
				} else {
					board.client.Out <- []byte(fmt.Sprintf("j %d %s", joinGameID, strings.ReplaceAll(joinGamePasswordField.GetText(), " ", "_")))
				}
				return nil
			} else if handleGameInput {
				inputMode = true
				buildLayout()
				return nil
			} else {
				selected := gameList.GetCurrentItemIndex()
				if viewScreen == ScreenLobby && len(allGames) > 0 && selected >= 0 && selected < len(allGames) {
					if allGames[selected].Password {
						showJoinGameDialog = true
						joinGameID = allGames[selected].ID
						buildLayout()
					} else {
						board.client.Out <- []byte(fmt.Sprintf("j %d", allGames[selected].ID))
					}
				}
				return nil
			}
		case tcell.KeyESC:
			if viewScreen == ScreenLobby {
				if showCreateGameDialog {
					showCreateGameDialog = false
					buildLayout()
					return nil
				} else if showJoinGameDialog {
					showJoinGameDialog = false
					buildLayout()
					return nil
				} else if !gameInProgress {
					return nil
				}
			} else if inputMode {
				inputMode = false
				buildLayout()
				return nil
			}
			newScreen := ScreenLobby
			if viewScreen == ScreenLobby {
				newScreen = ScreenGame
			}
			setScreen(newScreen)
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if inputMode {
				return event
			}

			// Undo move.
			if handleGameInput && len(board.Board.Moves) > 0 {
				lastMove := board.Board.Moves[len(board.Board.Moves)-1]
				board.client.Out <- []byte(fmt.Sprintf("mv %d/%d", lastMove[1], lastMove[0]))
				return nil
			}
		case tcell.KeyRune:
			if inputMode {
				return event
			} else if event.Rune() == '/' {
				inputMode = true
				if len(strings.TrimSpace(inputField.GetText())) == 0 {
					inputField.SetText("/")
				}
				buildLayout()
				return nil
			}

			if viewScreen == ScreenGame && !inputMode {
				if gameInProgress {
					switch event.Rune() {
					case 'r', 'R':
						if board.Board.MayRoll() {
							board.client.Out <- []byte("roll")
						}
					case 'k', 'K':
						if board.Board.MayOK() {
							board.client.Out <- []byte("ok")
						}
					}
				}
				return nil
			}
		}
		return event
	})

	loginForm.AddButton("Connect", connectFunc)
	loginForm.SetPadding(1, 0, 2, 0)

	logInHeader := cview.NewTextView()
	logInHeader.SetDynamicColors(true)
	logInHeader.SetText("[" + cview.ColorHex(cview.Styles.SecondaryTextColor) + "]Connect to bgammon.org[-]")

	logInFooter := cview.NewTextView()
	logInFooter.SetText(loginFooterText)

	f2 := cview.NewFlex() // Login flex
	f2.SetDirection(cview.FlexRow)
	f2.AddItem(logInHeader, 1, 1, true)
	f2.AddItem(loginForm, 8, 1, true)
	f2.AddItem(logInFooter, 0, 1, true)

	statusWriter = &bufferWriter{Buffer: statusBuffer}
	gameWriter = &bufferWriter{Buffer: gameBuffer}

	gameList = cview.NewList()
	gameList.ShowSecondaryText(false)
	gameList.SetHighlightFullLine(true)
	gameList.AddContextItem("Join", 'j', func(index int) {

	})
	gameList.SetSelectedFunc(func(i int, item *cview.ListItem) {
		if i < 0 || i >= len(allGames) {
			return
		}
		// TODO prompt for password when required
		entry := allGames[i]
		if entry.Password {
			showJoinGameDialog = true
			joinGameID = entry.ID
			buildLayout()
		} else {
			board.client.Out <- []byte(fmt.Sprintf("j %d", entry.ID))
		}
	})

	gameListHeader := cview.NewTextView()
	gameListHeader.SetText("[" + colorYellow + "]Status   Points   Name[-:-:-]")
	gameListHeader.SetDynamicColors(true)

	gameListFooter = cview.NewTextView()
	gameListFooter.SetDynamicColors(true)
	gameListFooter.SetRegions(true)
	gameListFooter.SetHighlightedFunc(func(added, removed, remaining []string) {
		defer gameListFooter.Highlight()
		if len(added) > 0 {
			switch added[0] {
			case "btncreate":
				showCreateGameDialog = true
				resetCreateGameDialog()
				buildLayout()
			case "btnrefresh":
				board.client.Out <- []byte("ls")
			case "btnautorefresh":
				autoRefresh = !autoRefresh
				buildLayout()
			}
		}
	})

	createGameLabel := cview.NewTextView()
	createGameLabel.SetDynamicColors(true)
	createGameLabel.SetText("[" + colorYellow + "]Create match[-]")

	publicOption := cview.NewDropDownOption("Public")
	publicOption.SetSelectedFunc(func(index int, option *cview.DropDownOption) {
		if createGamePasswordField == nil {
			return
		}
		createGamePasswordField.SetVisible(false)
	})

	privateOption := cview.NewDropDownOption("Private")
	privateOption.SetSelectedFunc(func(index int, option *cview.DropDownOption) {
		if createGamePasswordField == nil {
			return
		}
		createGamePasswordField.SetVisible(true)
	})

	createGameForm = cview.NewForm()
	createGameForm.AddInputField("Name", "", 10, nil, nil)
	createGameForm.AddInputField("Points", "", 10, func(textToCheck string, lastChar rune) bool {
		allowed := []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
		for _, allowedRune := range allowed {
			if lastChar == allowedRune {
				return true
			}
		}
		return false
	}, nil)
	createGameForm.AddDropDown("Type", 0, nil, []*cview.DropDownOption{publicOption, privateOption})
	createGameForm.AddPasswordField("Password", "", 10, '*', nil)
	createGameForm.AddButton("Cancel", func() {
		showCreateGameDialog = false
		buildLayout()
	})
	createGameForm.AddButton("Create", func() {
		acceptCreateGameDialog()
	})
	createGamePointsField = createGameForm.GetFormItem(1).(*cview.InputField)
	createGamePasswordField = createGameForm.GetFormItem(3).(*cview.InputField)
	resetCreateGameDialog()

	createGameGrid = cview.NewGrid()
	createGameGrid.SetRows(1, -1)
	createGameGrid.SetColumns(1, -1)
	createGameGrid.AddItem(createGameLabel, 0, 0, 1, 2, 0, 0, false)
	createGameGrid.AddItem(cview.NewBox(), 1, 0, 1, 1, 0, 0, true)
	createGameGrid.AddItem(createGameForm, 1, 1, 1, 1, 0, 0, true)

	joinGameLabelField = cview.NewTextView()
	joinGameLabelField.SetDynamicColors(true)
	joinGameLabelField.SetText("[" + colorYellow + "]Join match: " + cview.Escape(joinGameName) + "[-]")

	joinGameForm = cview.NewForm()
	joinGameForm.AddPasswordField("Password", "", 10, '*', nil)
	joinGameForm.AddButton("Cancel", func() {
		showJoinGameDialog = false
		buildLayout()
	})
	joinGameForm.AddButton("Join", func() {
		board.client.Out <- []byte(fmt.Sprintf("j %d %s", joinGameID, strings.ReplaceAll(joinGamePasswordField.GetText(), " ", "_")))
	})
	joinGamePasswordField = joinGameForm.GetFormItem(0).(*cview.InputField)

	joinGameGrid = cview.NewGrid()
	joinGameGrid.SetRows(1, -1)
	joinGameGrid.SetColumns(1, -1)
	joinGameGrid.AddItem(joinGameLabelField, 0, 0, 1, 2, 0, 0, false)
	joinGameGrid.AddItem(cview.NewBox(), 1, 0, 1, 1, 0, 0, true)
	joinGameGrid.AddItem(joinGameForm, 1, 1, 1, 1, 0, 0, true)

	gameListGrid = cview.NewGrid()
	gameListGrid.SetRows(1, -1, 1)
	gameListGrid.AddItem(gameListHeader, 0, 0, 1, 1, 0, 0, false)
	gameListGrid.AddItem(gameList, 1, 0, 1, 1, 0, 0, true)
	gameListGrid.AddItem(gameListFooter, 2, 0, 1, 1, 0, 0, false)

	uiGrid = cview.NewGrid()

	app.SetAfterResizeFunc(func(width int, height int) {
		screenWidth = width
		buildLayout()
	})

	buildLayout()
	defer func() {
		if c.Username != "" {
			app.SetRoot(uiGrid, true)
			app.SetFocus(inputField)
		}
	}()

	go HandleEvents(c, b)

	lg("This client and the bgammon.org server are free and open source:")
	lg("- https://code.rocket9labs.com/tslocum/bgammon")
	lg("- https://code.rocket9labs.com/tslocum/bgammon-cli")
	lg("Press <Enter> to enable text input.")

	if c.Username == "" {
		app.SetRoot(f2, true)
		app.SetFocus(loginForm)
	} else {
		logIn(c)
	}

	return app.Run()
}

func UpdateGameList(ev *bgammon.EventList) {
	allGames = make([]bgammon.GameListing, len(ev.Games))
	copy(allGames, ev.Games)

	gameList.Clear()

	if len(ev.Games) == 0 {
		gameList.AddItem(cview.NewListItem("*** No matches available. Please create one. ***"))
	} else {
		var entryStatus string
		for _, entry := range ev.Games {
			if entry.Players == 2 {
				entryStatus = "Full"
			} else {
				if !entry.Password {
					entryStatus = "Open"
				} else {
					entryStatus = "Private"
				}
			}
			gameList.AddItem(cview.NewListItem(fmt.Sprintf("%-7s  %-7s  %-30s", entryStatus, strconv.Itoa(int(entry.Points)), entry.Name)))
		}
	}

	if viewScreen == ScreenLobby && !showCreateGameDialog && !showJoinGameDialog {
		app.Draw()
	}
}

func HandleEvents(c *Client, b *GameBoard) {
	for e := range c.Events {
		switch ev := e.(type) {
		case *bgammon.EventWelcome:
			c.Username = ev.PlayerName
			areIs := "are"
			if ev.Clients == 1 {
				areIs = "is"
			}
			clientsPlural := "s"
			if ev.Clients == 1 {
				clientsPlural = ""
			}
			matchesPlural := "es"
			if ev.Games == 1 {
				matchesPlural = ""
			}
			l(fmt.Sprintf("*** Welcome, %s. There %s %d client%s playing %d match%s.", ev.PlayerName, areIs, ev.Clients, clientsPlural, ev.Games, matchesPlural))
		case *bgammon.EventHelp:
			l(fmt.Sprintf("Help: %s", ev.Message))
		case *bgammon.EventPing:
			c.Out <- []byte(fmt.Sprintf("pong %s", ev.Message))
		case *bgammon.EventNotice:
			l(fmt.Sprintf("*** %s", ev.Message))
		case *bgammon.EventSay:
			l(fmt.Sprintf("<%s> %s", ev.Player, ev.Message))
		case *bgammon.EventList:
			UpdateGameList(ev)
		case *bgammon.EventJoined:
			if ev.PlayerNumber == 1 {
				b.Board.Player1.Name = ev.Player
			} else if ev.PlayerNumber == 2 {
				b.Board.Player2.Name = ev.Player
			}
			b.Update()

			gameInProgress = true
			showCreateGameDialog = false
			showJoinGameDialog = false
			setScreen(ScreenGame)

			if ev.Player == c.Username {
				gameBuffer.SetText("")
				gameLogged = false
			} else {
				lg(fmt.Sprintf("%s joined the match.", ev.Player))
			}
		case *bgammon.EventFailedJoin:
			l(fmt.Sprintf("*** Failed to join match: %s", ev.Reason))
		case *bgammon.EventLeft:
			if b.Board.Player1.Name == ev.Player {
				b.Board.Player1.Name = ""
			} else if b.Board.Player2.Name == ev.Player {
				b.Board.Player2.Name = ""
			}
			b.Update()
			if ev.Player == c.Username {
				gameInProgress = false
				setScreen(ScreenLobby)
			} else {
				lg(fmt.Sprintf("%s left the match.", ev.Player))
			}
		case *bgammon.EventBoard:
			b.Board = &ev.GameState
			b.Update()
		case *bgammon.EventRolled:
			b.Board.Roll1 = ev.Roll1
			b.Board.Roll2 = ev.Roll2
			var diceFormatted string
			if b.Board.Turn == 0 {
				if b.Board.Player1.Name == ev.Player {
					diceFormatted = fmt.Sprintf("%d", b.Board.Roll1)
				} else {
					diceFormatted = fmt.Sprintf("%d", b.Board.Roll2)
				}
			} else {
				diceFormatted = fmt.Sprintf("%d-%d", b.Board.Roll1, b.Board.Roll2)
			}
			b.Update()
			lg(fmt.Sprintf("%s rolled %s.", ev.Player, diceFormatted))
		case *bgammon.EventFailedRoll:
			l(fmt.Sprintf("*** Failed to roll: %s", ev.Reason))
		case *bgammon.EventMoved:
			b.Update()
			lg(fmt.Sprintf("%s moved %s.", ev.Player, bgammon.FormatMoves(ev.Moves)))
		case *bgammon.EventFailedMove:
			c.Out <- []byte("board") // Refresh game state.

			var extra string
			if ev.From != 0 || ev.To != 0 {
				extra = fmt.Sprintf(" from %s to %s", bgammon.FormatSpace(ev.From), bgammon.FormatSpace(ev.To))
			}
			l(fmt.Sprintf("*** Failed to move checker%s: %s", extra, ev.Reason))
			l(fmt.Sprintf("*** Legal moves: %s", bgammon.FormatMoves(b.Board.Available)))
		case *bgammon.EventFailedOk:
			c.Out <- []byte("board") // Refresh game state.
			l(fmt.Sprintf("*** Failed to submit moves: %s", ev.Reason))
		case *bgammon.EventWin:
			lg(fmt.Sprintf("%s wins!", ev.Player))
		case *bgammon.EventSettings:
			// Do nothing.
		default:
			l(fmt.Sprintf("*** WARNING: You may need to upgrade your client. Unknown event received: %+v %+v", ev, e))
		}
	}
}
