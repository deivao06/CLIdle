package ui

import (
	"context"
	"time"

	"clidle/internal/lang"
	"clidle/internal/update"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	updateCheckEvery = time.Hour
	updateInstallMax = 2 * time.Minute
	updateErrShow    = 8 * time.Second
)

type updateFoundMsg struct{ r *update.Release }

type updateDoneMsg struct{ err error }

func checkUpdate() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), update.Timeout)
		defer cancel()
		r, _ := update.Check(ctx)
		return updateFoundMsg{r}
	}
}

// installUpdate asks GitHub once more, since a newer release may have come out since the notice.
func installUpdate(found *update.Release) tea.Cmd {
	return func() tea.Msg {
		if update.Site != "" {
			return updateDoneMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), updateInstallMax)
		defer cancel()
		r := found
		if latest, err := update.Check(ctx); err == nil && latest != nil {
			r = latest
		}
		exe, err := update.Executable()
		if err != nil {
			return updateDoneMsg{err}
		}
		return updateDoneMsg{update.Install(ctx, r, exe)}
	}
}

// townArrival runs on live ticks: the first tick and reaching town look for an update, at most once
// an hour, and stop looking once one was found.
func (m *Model) townArrival(now time.Time) tea.Cmd {
	inTown := m.g.InTown()
	arrived := inTown && !m.wasInTown
	m.wasInTown = inTown
	switch {
	case !arrived && (!m.lastCheck.IsZero() || m.checking):
		return nil
	case m.release != nil:
		if !m.g.Auto {
			m.askUpdate()
		}
		return nil
	case !update.Enabled() || m.checking || !m.lastCheck.IsZero() && now.Sub(m.lastCheck) < updateCheckEvery:
		return nil
	}
	m.checking, m.lastCheck = true, now
	return checkUpdate()
}

func (m *Model) updateFound(r *update.Release) {
	m.checking = false
	if r == nil || m.catching {
		return
	}
	m.release = r
	if m.g.InTown() && !m.g.Auto {
		m.askUpdate()
	}
}

func (m *Model) askUpdate() {
	if m.release == nil || m.updating || m.confirmDo != nil || m.summary != nil || m.choosing || m.naming {
		return
	}
	m.ask(lang.T("confirm.update", m.release.Version, update.Version), func(m *Model) {
		m.updating = true
		m.cmd = installUpdate(m.release)
	})
	m.confirmKeep = true
}

func (m *Model) updateDone(err error) tea.Cmd {
	m.updating = false
	if err != nil {
		m.updateErr, m.updateErrAt = lang.T("update.failed", err.Error()), time.Now()
		return nil
	}
	m.restart = true
	m.save(time.Now())
	return tea.Quit
}

// Restarting reports that the binary was updated and the program should start it again.
func (m Model) Restarting() bool { return m.restart }

func (m Model) updateNotice() string {
	switch {
	case m.updating:
		return sWarn.Render(lang.T("update.updating")) + "  "
	case m.release != nil:
		return sWarn.Render(lang.T("update.notice", m.release.Version)) + "  "
	}
	return ""
}
