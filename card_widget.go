package main

import (
	"strconv"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// this required to go generate to work
// https://github.com/cogentcore/core/discussions/1243
var _ tree.Node = nil


type Card struct {
	core.Frame
	Heading    string
	SubHeading string
	Data       interface{}
}

func (cd *Card) Init() {
	cd.Frame.Init()
	cd.Frame.OnClick(cd.onClickEvent)

	tree.AddChild(cd, func(w *core.Text) {
		w.SetType(core.TextHeadlineSmall)
		w.Updater(func() {
			w.SetText(cd.Heading)
		})
		w.Styler(func (s *styles.Style) {
			s.SetNonSelectable()
			s.Font.Weight = styles.WeightBold
		})
	})

	tree.AddChild(cd, func(w *core.Text) {
		w.Updater(func() {
			w.SetText(cd.SubHeading)
		})
		w.Styler(func (s *styles.Style) {
			s.SetNonSelectable()
		})
	})

	cd.Styler(func (s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
		s.Border.Width.Set(units.Dp(4))
		s.Border.Color.Set(colors.Scheme.Outline)
		s.SetAbilities(true, abilities.Selectable)
		s.Cursor = cursors.Pointer
	})
}

func (cd *Card) showSelection() {
	vTimer := time.NewTimer(100 * time.Millisecond)
	<-vTimer.C
	cd.AsyncLock()
	cd.SetState(false, states.Focused)
	cd.Update()
	cd.AsyncUnlock()
}

func (cd *Card) onClickEvent(e events.Event) {
	cd.SetState(true, states.Focused)
	cd.Update()
	go cd.showSelection()
}
// ----------------------------------------------------------------------

type CardData struct {
	Data       any
	Heading    string
	SubHeading string
}

type CardList struct {
	core.Frame
	OnClick       func(v CardData)
	cardDataSlice []CardData
}

func (this *CardList) Init() {
	this.Frame.Init()

	this.Frame.Styler(func (s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowAuto)
	})

	this.Frame.Updater(func() {
		this.Frame.ScrollDimToContentStart(math32.Y)
	})

	this.Frame.Maker(func(p *tree.Plan) {
		for i := range this.cardDataSlice {
			tree.AddAt(p, strconv.Itoa(i), func(w *Card) {
				w.Updater(func () {
					vData := this.cardDataSlice[i]
					w.SetHeading(vData.Heading)
					w.SetSubHeading(vData.SubHeading)
					w.SetData(vData.Data)
				})
				w.OnClick(func(e events.Event) {
					this.OnClick(this.cardDataSlice[i])
				})
			})
		}
	})
}

func (this *CardList) Add(pId any, pHeading string, pSubHeading string) {
	this.cardDataSlice = append(this.cardDataSlice, CardData{Data: pId, Heading: pHeading, SubHeading: pSubHeading})
}

func (this *CardList) Length() int {
	return len(this.cardDataSlice)
}

func (this *CardList) Clear() {
	this.cardDataSlice = nil
	this.Update()
}
// ----------------------------------------------------------------------
