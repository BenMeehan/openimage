package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anomalyco/openimage/pkg/config"
	"github.com/anomalyco/openimage/pkg/display"
	"github.com/anomalyco/openimage/pkg/provider"
)

type tuiState int

const (
	statePrompt tuiState = iota
	stateGenerating
	stateDone
	stateError
)

var (
	bg   = lipgloss.Color("#0f0f11")
	acc  = lipgloss.Color("#c084fc")
	sub  = lipgloss.Color("#27272a")
	mut  = lipgloss.Color("#a1a1aa")
	good = lipgloss.Color("#4ade80")
	bad  = lipgloss.Color("#f87171")
	text = lipgloss.Color("#fafafa")

	pad = lipgloss.NewStyle().Padding(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(acc).
			Background(bg).
			PaddingTop(1).
			Align(lipgloss.Center)

	heroBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(acc).
			Background(bg).
			Padding(2, 4)

	doneBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(good).
			Background(bg).
			Padding(1, 2)

	errBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(bad).
			Background(bg).
			Padding(1, 2)

	inputStyle = lipgloss.NewStyle().
			Width(58).
			Background(bg)

	helpStyle  = lipgloss.NewStyle().Foreground(mut).Background(bg)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525b")).Background(bg)
	spinnerAcc = lipgloss.NewStyle().Foreground(acc).Background(bg)
	doneLabel  = lipgloss.NewStyle().Foreground(good).Background(bg)

	pbarStyle = progress.New(
		progress.WithGradient("#c084fc", "#7c3aed"),
		progress.WithoutPercentage(),
	)
)

type tuiModel struct {
	prov   provider.Provider
	cfg    *config.Config
	key    string

	state   tuiState
	prompt  textinput.Model
	result  []byte
	outPath string
	errMsg  string
	width   int
	height  int

	progress progress.Model
	pbarPct  float64
}

func runTUI(prov provider.Provider, key string, cfg *config.Config) error {
	ti := textinput.New()
	ti.Placeholder = "a cinematic portrait of a robot reading in a library..."
	ti.CharLimit = 2000
	ti.Width = 58
	ti.Focus()
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(acc).Background(bg)
	ti.TextStyle = lipgloss.NewStyle().Foreground(text).Background(bg)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525b")).Background(bg)

	m := &tuiModel{
		prov:     prov,
		cfg:      cfg,
		key:      key,
		state:    statePrompt,
		prompt:   ti,
		progress: pbarStyle,
	}

	program := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func (m *tuiModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if m.state == stateGenerating {
			m.pbarPct += 0.03
			if m.pbarPct > 0.95 {
				m.pbarPct = 0.95
			}
			return m, tickCmd()
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case statePrompt:
			return m.handlePromptInput(msg)
		case stateDone, stateError:
			switch msg.String() {
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			case "enter", "n":
				m.state = statePrompt
				m.prompt.Reset()
				m.prompt.Focus()
				m.result = nil
				m.errMsg = ""
				m.pbarPct = 0
				return m, textinput.Blink
			}
		}

	case generationResult:
		m.state = stateDone
		m.result = msg.data
		m.outPath = msg.path
		m.pbarPct = 1.0
		return m, nil

	case generationError:
		m.state = stateError
		m.errMsg = msg.err.Error()
		return m, nil
	}

	if m.state == statePrompt {
		var cmd tea.Cmd
		m.prompt, cmd = m.prompt.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *tuiModel) handlePromptInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		prompt := strings.TrimSpace(m.prompt.Value())
		if prompt == "" {
			return m, nil
		}
		m.state = stateGenerating
		m.pbarPct = 0
		return m, tea.Batch(tickCmd(), m.generateImage(prompt))
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m *tuiModel) generateImage(prompt string) tea.Cmd {
	return func() tea.Msg {
		img, err := m.prov.GenerateImage(&provider.GenerateParams{
			Prompt:  prompt,
			Model:   m.cfg.Model,
			N:       1,
			Size:    m.cfg.Size,
			Quality: m.cfg.Quality,
		})
		if err != nil {
			return generationError{err}
		}
		dir := m.cfg.SaveDir
		if dir == "" {
			dir = "."
		}
		os.MkdirAll(dir, 0755)
		outPath := resolveOutputPath(0, 1)
		os.WriteFile(outPath, img.Data, 0644)
		return generationResult{data: img.Data, path: outPath}
	}
}

func keyHint(keys string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Background(bg).Bold(true).Render(keys)
}

func (m *tuiModel) View() string {
	header := titleStyle.Width(m.width).Render("✧  openimage")
	footer := m.viewFooter()

	var body string
	switch m.state {
	case statePrompt:
		body = m.viewPrompt()
	case stateGenerating:
		body = m.viewGenerating()
	case stateDone:
		body = m.viewDone()
	case stateError:
		body = m.viewError()
	}

	bodyH := lipgloss.Height(body)
	gap := m.height - lipgloss.Height(header) - bodyH - lipgloss.Height(footer) - 2
	if gap < 0 {
		gap = 0
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		strings.Repeat("\n", gap/2),
		body,
		strings.Repeat("\n", gap-gap/2),
		footer,
	)

	return lipgloss.NewStyle().
		Width(m.width).Height(m.height).
		Background(bg).
		Render(pad.Render(content))
}

func (m *tuiModel) viewPrompt() string {
	input := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(acc).
		Background(sub).
		Padding(1, 2).
		Render(inputStyle.Render(m.prompt.View()))

	return lipgloss.JoinVertical(lipgloss.Center,
		helpStyle.Render("describe the image you want to generate"),
		"",
		input,
	)
}

func (m *tuiModel) viewGenerating() string {
	pbar := m.progress.ViewAs(m.pbarPct)
	spin := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	ch := string(spin[int(time.Now().UnixMilli()/80)%len(spin)])

	inner := lipgloss.JoinVertical(lipgloss.Center,
		spinnerAcc.Render(ch+"  generating..."),
		"",
		lipgloss.NewStyle().Width(50).Render(pbar),
		"",
		dimStyle.Render(truncate(m.prompt.Value(), 50)),
	)

	return heroBox.Render(inner)
}

func (m *tuiModel) viewDone() string {
	img := display.ShowInline(m.result)

	status := lipgloss.JoinHorizontal(lipgloss.Left,
		doneLabel.Render("●  saved"),
		helpStyle.Render("  ·  "+m.outPath),
		helpStyle.Render(fmt.Sprintf("  ·  %s", formatBytes(len(m.result)))),
	)

	inner := lipgloss.JoinVertical(lipgloss.Center, status, "", img)
	return doneBox.Render(inner)
}

func (m *tuiModel) viewError() string {
	return errBox.Render(
		lipgloss.NewStyle().Foreground(bad).Background(bg).Render("✗  "+m.errMsg),
	)
}

func (m *tuiModel) viewFooter() string {
	enter  := lipgloss.NewStyle().Foreground(good).Background(bg).Bold(true).Render("enter")
	ctrlC  := lipgloss.NewStyle().Foreground(bad).Background(bg).Bold(true).Render("ctrl+c")
	qKey   := lipgloss.NewStyle().Foreground(bad).Background(bg).Bold(true).Render("q")

	var left, right string
	switch m.state {
	case statePrompt:
		left  = enter + helpStyle.Render("  generate")
		right = ctrlC + helpStyle.Render("  quit")
	case stateDone:
		left  = enter + helpStyle.Render("  new image")
		right = qKey + helpStyle.Render("  quit")
	case stateError:
		left  = enter + helpStyle.Render("  try again")
		right = qKey + helpStyle.Render("  quit")
	default:
		right = ctrlC + helpStyle.Render("  cancel")
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Width(m.width/2-4).Align(lipgloss.Right).Background(bg).Render(left),
		lipgloss.NewStyle().Width(8).Background(bg).Render(""),
		lipgloss.NewStyle().Width(m.width/2-4).Align(lipgloss.Left).Background(bg).Render(right),
	)
}

func formatBytes(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%dKB", n/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(n)/(1024*1024))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

type generationResult struct {
	data []byte
	path string
}

type generationError struct {
	err error
}
