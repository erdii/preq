package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/erdii/preq/internal/querycmd"
)

const (
	colorActive   = tcell.ColorWhite
	colorInactive = tcell.ColorDarkGray
	queryTimeout  = 5 * time.Second
)

// TODO: make configurable to support different query tools
// TODO: add `inputBuilder` to eg optionally process data without query (and get syntax highlighting)
var (
	successBuilder = querycmd.NewBuilder("yq", "{+q}")
	regularBuilder = querycmd.NewBuilder("yq", "-CP", "{+q}")
)

func initialQuery() string {
	if len(os.Args) > 1 {
		return strings.Join(os.Args[1:], " ")
	}

	return "."
}

func main() {
	if !stdinIsPipe() {
		fmt.Fprintln(os.Stderr, "stdin must be a pipe")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := tview.NewApplication()

	inputView := tview.NewTextView().
		SetDynamicColors(true)

	outputView := tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	inputView.SetBorder(true).
		SetBorderColor(colorActive)
	outputView.SetBorder(true).
		SetBorderColor(colorInactive)

	// redirect stdin into input view
	go func() {
		if _, err := io.Copy(io.MultiWriter(
			tview.ANSIWriter(inputView),
		), os.Stdin); err != nil {
			panic(err)
		}
	}()

	queryView := tview.NewTextArea().
		SetSize(1, 0).
		SetText(initialQuery(), true)

	outputWriter := tview.ANSIWriter(outputView)

	renderQueryAsync := func() {
		go func() {
			result, err := querycmd.Execute(ctx, regularBuilder, queryView.GetText(), inputView.GetText(false))
			if err != nil {
				// TODO: ignore for now
			}

			app.QueueUpdate(func() {
				outputView.Clear()
				fmt.Fprint(outputWriter, result)
			})
		}()
	}

	// rerun query whenever either the data input or the query input changes
	queryView.SetChangedFunc(renderQueryAsync)
	inputView.SetChangedFunc(func() {
		app.Draw()
		renderQueryAsync()
	})

	success := false

	// handle special keys and text navigation not meant for query input
	queryView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		// press escape, ctrl+c or ctrl+d to exit immediately
		case tcell.KeyEscape, tcell.KeyCtrlC, tcell.KeyCtrlD:
			app.Stop()
			return nil

		// press enter to exit and print resulting document to stdout and query to stderr
		case tcell.KeyEnter:
			success = true
			app.Stop()
			return nil

		// press tab to toggle active text view
		case tcell.KeyTab:
			if inputView.GetBorderColor() == colorActive {
				inputView.SetBorderColor(colorInactive)
				outputView.SetBorderColor(colorActive)
			} else {
				outputView.SetBorderColor(colorInactive)
				inputView.SetBorderColor(colorActive)
			}
			return nil

		// press arrows, pgup/down and home/end to navigate active text view
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
			if inputView.GetBorderColor() == colorActive {
				inputView.InputHandler()(event, func(p tview.Primitive) {})
			} else {
				outputView.InputHandler()(event, func(p tview.Primitive) {})
			}
			return nil
		}

		// all other keypresses are passed through to the query text view
		return event
	})

	// application layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(tview.NewTextView().
				SetSize(1, 5).
				SetTextAlign(tview.AlignRight).
				SetText("query"),
				0, 1, false).
			AddItem(queryView, 0, 1, true),
			1, 0, true).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(inputView, 0, 1, false).
			AddItem(outputView, 0, 1, false),
			0, 1, false)

	// run ui loop - blocks until shutdown
	if err := app.SetRoot(flex, true).SetFocus(queryView).Run(); err != nil {
		panic(err)
	}

	// shutdown handler
	if success {
		query := queryView.GetText()
		data := inputView.GetText(false)

		result, err := querycmd.Execute(ctx, successBuilder, query, data)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(os.Stdout, result)
		fmt.Fprintln(os.Stderr, query)
	}
}

func stdinIsPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	return stat.Mode()&os.ModeCharDevice == 0
}
