package ui

import (
	"context"
	"strings"
	"time"

	"clidle/internal/lang"
	"clidle/internal/update"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	newsPage  = "help.page.news"
	newsRetry = 10 * time.Second
)

type newsMsg struct {
	notes []update.Note
	err   error
}

func fetchNews() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), update.Timeout)
		defer cancel()
		notes, err := update.Notes(ctx)
		return newsMsg{notes, err}
	}
}

// wantsNews is true when an existing save runs a version whose notes it has not opened yet.
func wantsNews(seen string, fresh bool) bool {
	return !fresh && update.Version != "dev" && seen != update.Version
}

// newsCmd loads the notes when the news page is on screen and they are missing.
func (m *Model) newsCmd() tea.Cmd {
	if !m.help || helpPages[m.helpPage] != newsPage || m.news != nil || m.newsLoading || time.Since(m.newsAt) < newsRetry {
		return nil
	}
	m.newsLoading = true
	return fetchNews()
}

func (m *Model) newsArrived(msg newsMsg) {
	m.newsLoading, m.newsAt = false, time.Now()
	m.news, m.newsErr = msg.notes, msg.err
	if msg.err != nil || !m.newsAuto {
		return
	}
	m.newsAuto, m.newsSeen = false, true
	if !m.choosing {
		m.help, m.helpPage, m.helpTop = true, newsPageIndex(), 0
	}
}

func newsPageIndex() int {
	for i, p := range helpPages {
		if p == newsPage {
			return i
		}
	}
	return 0
}

var mdStrip = strings.NewReplacer("**", "", "`", "")

// newsText turns the release notes markdown into help page lines.
func (m Model) newsText() string {
	switch {
	case m.news != nil:
	case m.newsErr != nil:
		return lang.T("help.news.error")
	default:
		return lang.T("help.news.loading")
	}
	var b strings.Builder
	b.WriteString(lang.T("help.news.intro"))
	for _, n := range m.news {
		b.WriteString("\n\n# " + n.Version + "  " + n.Date)
		for _, l := range strings.Split(strings.ReplaceAll(n.Body, "\r", ""), "\n") {
			l = mdStrip.Replace(strings.TrimSpace(l))
			switch {
			case l == "" || strings.HasPrefix(l, "Commits:"):
				continue
			case strings.HasPrefix(l, "#"):
				l = "## " + strings.TrimLeft(l, "# ")
			case strings.HasPrefix(l, "* ") || strings.HasPrefix(l, "- "):
				l = "  •  " + l[2:]
			case strings.HasPrefix(l, "> "):
				l = "  " + l[2:]
			}
			b.WriteString("\n" + l)
		}
	}
	return b.String()
}
