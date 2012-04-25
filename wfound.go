package uik

import (
	"github.com/skelterjohn/geom"
	"github.com/skelterjohn/go.wde"
	"image/draw"
	"time"
)

const FrameDelay = 15 * time.Millisecond

// A foundation that wraps a wde.Window
type WindowFoundation struct {
	Foundation
	W               wde.Window
	Pane            *Block
	waitForRepaint  chan bool
	doRepaintWindow chan bool
}

func NewWindow(parent wde.Window, width, height int) (wf *WindowFoundation, err error) {
	wf = new(WindowFoundation)

	wf.W, err = WindowGenerator(parent, width, height)
	if err != nil {
		return
	}
	wf.Size = geom.Coord{float64(width), float64(height)}
	wf.Initialize()
	// Report(wf.ID, "is window")

	go wf.handleWindowEvents()
	go wf.handleWindowDrawing()
	go wf.HandleEvents()

	return
}

func (wf *WindowFoundation) Initialize() {
	wf.Foundation.Initialize()

	wf.DrawOp = draw.Src

	wf.waitForRepaint = make(chan bool)
	wf.doRepaintWindow = make(chan bool)
	wf.Invalidations = make(chan Invalidation, 1)

	wf.Paint = ClearPaint

	// Report("wfound is", wf.ID)

	wf.HasKeyFocus = true
}

func (wf *WindowFoundation) SetPane(b *Block) {
	if wf.Pane != nil {
		wf.RemoveBlock(wf.Pane)
	}
	wf.Pane = b
	// Report("pane", wf.ID, b.ID)
	wf.PlaceBlock(b, geom.Rect{geom.Coord{}, wf.Size})
}

func (wf *WindowFoundation) Show() {
	wf.W.Show()
	wf.Invalidate()
}

func (wf *WindowFoundation) HandleEvent(e interface{}) {
	switch e := e.(type) {
	case ResizeEvent:
		wf.DoResizeEvent(e)
		if wf.Pane != nil {
			wf.Pane.UserEventsIn.SendOrDrop(e)
		}
		wf.ChildrenBounds[wf.Pane] = geom.Rect{geom.Coord{}, e.Size}
	default:
		wf.Foundation.HandleEvent(e)
	}
}

// dispense events to children, as appropriate
// func (wf *WindowFoundation) HandleEvents() {
// 	for {
// 		select {
// 		case e := <-wf.UserEvents:
// 			wf.HandleEvent(e)
// 		case e := <-wf.BlockInvalidations:
// 			wf.DoBlockInvalidation(e)
// 		}
// 	}
// }

// wraps mouse events with float64 coordinates
func (wf *WindowFoundation) handleWindowEvents() {
	for e := range wf.W.EventChan() {
		ev := Event{
			TimeSinceStart(),
		}
		switch e := e.(type) {
		case wde.CloseEvent:
			wf.UserEventsIn.SendOrDrop(CloseEvent{
				Event:      ev,
				CloseEvent: e,
			})
		case wde.MouseDownEvent:
			wf.UserEventsIn.SendOrDrop(MouseDownEvent{
				Event:          ev,
				MouseDownEvent: e,
				MouseLocator: MouseLocator{
					Loc: geom.Coord{float64(e.Where.X), float64(e.Where.Y)},
				},
			})
		case wde.MouseUpEvent:
			// Report("wde.MouseUpEvent")
			wf.UserEventsIn.SendOrDrop(MouseUpEvent{
				Event:        ev,
				MouseUpEvent: e,
				MouseLocator: MouseLocator{
					Loc: geom.Coord{float64(e.Where.X), float64(e.Where.Y)},
				},
			})
		case wde.KeyDownEvent:
			wf.UserEventsIn.SendOrDrop(KeyDownEvent{
				Event:        ev,
				KeyDownEvent: e,
			})
		case wde.KeyUpEvent:
			wf.UserEventsIn.SendOrDrop(KeyUpEvent{
				Event:      ev,
				KeyUpEvent: e,
			})
		case wde.KeyTypedEvent:
			wf.UserEventsIn.SendOrDrop(KeyTypedEvent{
				Event:         ev,
				KeyTypedEvent: e,
			})
		case wde.ResizeEvent:
			// Report(wf.ID, "wde.ResizeEvent")
			wf.waitForRepaint <- true
			wf.UserEventsIn.SendOrDrop(ResizeEvent{
				Event:       ev,
				ResizeEvent: e,
				Size: geom.Coord{
					X: float64(e.Width),
					Y: float64(e.Height),
				},
			})
			wf.Invalidate()
		}
	}
}

func (wf *WindowFoundation) SleepRepaint(delay time.Duration) {
	time.Sleep(delay)
	wf.doRepaintWindow <- true
}

func (wf *WindowFoundation) handleWindowDrawing() {

	waitingForRepaint := false
	newStuff := false

	flush := func() {
		wf.W.FlushImage()
		newStuff = false
		waitingForRepaint = true
		go wf.SleepRepaint(FrameDelay)
	}

	for {
		select {
		case <-wf.Invalidations:
			// Report("window invalidation")
			if waitingForRepaint {
				newStuff = true
			} else {
				waitingForRepaint = true
				newStuff = true
				go wf.SleepRepaint(FrameDelay)
			}
		case waitingForRepaint = <-wf.waitForRepaint:
		case <-wf.doRepaintWindow:
			waitingForRepaint = false
			if !newStuff {
				break
			}
			wf.Pane.Drawer.Draw(wf.W.Screen())
			flush()
		}
	}
}
