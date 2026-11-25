package ui

// ScrollMsg scrolls by that many lines (positive = down); in lists it moves the cursor and the list follows it.
type ScrollMsg int

// scrolled is where the last View left the scrollable part: its first line and the last first line it can take.
type scrolled struct{ top, max int }

func (m *Model) scroll(n int) {
	if m.summary != nil || m.choosing || m.naming || m.confirmDo != nil {
		return
	}
	top := clamp(m.shown.top+n, 0, m.shown.max)
	switch {
	case m.help:
		m.helpTop = top
	case m.tab == tabHero:
		m.heroTop = top
	case m.listShown():
		m.cursor = clamp(m.cursor+n, 0, len(m.rows())-1)
	}
}
