package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Couleurs adaptatives ────────────────────────────────────────────────────
// Chaque couleur a une variante claire et une variante sombre ; lipgloss
// détecte le fond du terminal (via termenv) et choisit la bonne. On évite
// les grands aplats de fond : le thème du terminal reste maître.

var (
	accent = lipgloss.AdaptiveColor{Light: "#D6488C", Dark: "#FF9EC4"}
	okCol  = lipgloss.AdaptiveColor{Light: "#1F7A4D", Dark: "#7BD8A8"}
	koCol  = lipgloss.AdaptiveColor{Light: "#B3243C", Dark: "#FF7B8A"}
	muted  = lipgloss.AdaptiveColor{Light: "#8A7D8F", Dark: "#9A8CA0"}
	soft   = lipgloss.AdaptiveColor{Light: "#6E5A75", Dark: "#C9BBD1"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).Foreground(accent)

	gridBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2)

	cellStyle = lipgloss.NewStyle().
			Width(5).Height(1).Align(lipgloss.Center).Foreground(soft)

	cursorStyle = lipgloss.NewStyle().
			Width(5).Height(1).Align(lipgloss.Center).
			Reverse(true).Bold(true)

	hintStyle = lipgloss.NewStyle().
			Width(5).Height(1).Align(lipgloss.Center).
			Bold(true).Faint(true)

	sumWait = lipgloss.NewStyle(). // ligne/colonne incomplète
		Width(5).Align(lipgloss.Center).Foreground(muted)

	sumOK = lipgloss.NewStyle(). // complète et correcte
		Width(5).Align(lipgloss.Center).Bold(true).Foreground(okCol)

	sumKO = lipgloss.NewStyle(). // complète mais fausse
		Width(5).Align(lipgloss.Center).Bold(true).Foreground(koCol)

	palFree = lipgloss.NewStyle(). // chiffre encore disponible
		Width(3).Align(lipgloss.Center).Bold(true).Foreground(accent)

	palUsed = lipgloss.NewStyle(). // chiffre déjà posé
		Width(3).Align(lipgloss.Center).Foreground(muted)

	menuStyle = lipgloss.NewStyle().Foreground(soft)
	selStyle  = lipgloss.NewStyle().Bold(true).Foreground(accent)
	msgStyle  = lipgloss.NewStyle().Bold(true).Foreground(accent)
	helpStyle = lipgloss.NewStyle().Faint(true)
	pawStyle  = lipgloss.NewStyle().Foreground(accent)
)

// ─── Niveaux ─────────────────────────────────────────────────────────────────

type level struct {
	name  string
	hints int
}

var levels = []level{
	{"facile", 2},
	{"moyen", 1},
	{"difficile", 0},
}

// ─── Modèle ──────────────────────────────────────────────────────────────────

type model struct {
	menu     bool
	levelIdx int
	grid     [9]int // -1 = vide
	hintCell [9]bool
	used     [10]bool
	sol      [9]int
	rowSum   [3]int
	colSum   [3]int
	cursor   int
	won      bool
	message  string
	paws     int
	width    int
	height   int
}

func newModel() model {
	m := model{menu: true}
	m.width, m.height = 80, 24
	return m
}

func startGame(levelIdx, width, height int) model {
	m := model{levelIdx: levelIdx, width: width, height: height}
	for i := range m.grid {
		m.grid[i] = -1
	}
	for i := range m.sol {
		m.sol[i] = i + 1
	}
	rand.Shuffle(9, func(i, j int) { m.sol[i], m.sol[j] = m.sol[j], m.sol[i] })
	m.rowSum = [3]int{m.sol[0] + m.sol[1] + m.sol[2], m.sol[3] + m.sol[4] + m.sol[5], m.sol[6] + m.sol[7] + m.sol[8]}
	m.colSum = [3]int{m.sol[0] + m.sol[3] + m.sol[6], m.sol[1] + m.sol[4] + m.sol[7], m.sol[2] + m.sol[5] + m.sol[8]}
	// indices : chiffres corrects pré-placés, non effaçables
	n := levels[levelIdx].hints
	positions := rand.Perm(9)[:n]
	for _, p := range positions {
		m.grid[p] = m.sol[p]
		m.hintCell[p] = true
		m.used[m.sol[p]] = true
	}
	m.message = "Place les chiffres (๑˃ᴗ˂)ﻭ"
	return m
}

func (m model) rowSumOf(i int) int { return m.grid[3*i] + m.grid[3*i+1] + m.grid[3*i+2] }
func (m model) colSumOf(j int) int { return m.grid[j] + m.grid[3+j] + m.grid[6+j] }
func (m model) rowFull(i int) bool { return m.grid[3*i] > 0 && m.grid[3*i+1] > 0 && m.grid[3*i+2] > 0 }
func (m model) colFull(j int) bool { return m.grid[j] > 0 && m.grid[3+j] > 0 && m.grid[6+j] > 0 }
func (m model) full() bool {
	for i := 0; i < 3; i++ {
		if !m.rowFull(i) {
			return false
		}
	}
	return true
}
func (m model) checkWin() bool {
	for i := 0; i < 3; i++ {
		if m.rowSumOf(i) != m.rowSum[i] || m.colSumOf(i) != m.colSum[i] {
			return false
		}
	}
	return true
}

type tickMsg struct{}

const tickInterval = 400 * time.Millisecond

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		if m.won {
			m.paws++
		}
		return m, tick()
	case tea.KeyMsg:
		if m.menu {
			switch msg.String() {
			case "1", "2", "3":
				return startGame(int(msg.String()[0]-'1'), m.width, m.height), nil
			case "up", "k", "left", "h":
				if m.levelIdx > 0 {
					m.levelIdx--
				}
			case "down", "j", "right", "l":
				if m.levelIdx < len(levels)-1 {
					m.levelIdx++
				}
			case "enter", " ", "home":
				return startGame(m.levelIdx, m.width, m.height), nil
			case "q", "ctrl+c", "esc":
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "m":
			return newModel(), nil
		case "n":
			return startGame(m.levelIdx, m.width, m.height), nil
		case "up", "k":
			if m.cursor >= 3 {
				m.cursor -= 3
			}
		case "down", "j":
			if m.cursor < 6 {
				m.cursor += 3
			}
		case "left", "h":
			if m.cursor%3 > 0 {
				m.cursor--
			}
		case "right", "l":
			if m.cursor%3 < 2 {
				m.cursor++
			}
		case "backspace", "x", "0":
			v := m.grid[m.cursor]
			if v > 0 && !m.hintCell[m.cursor] {
				m.used[v] = false
				m.grid[m.cursor] = -1
				m.message = "Case vidée~"
			} else if m.hintCell[m.cursor] {
				m.message = "C'est un indice, il reste ! (｡•̀ᴗ-)✧"
			}
		default:
			if len(msg.String()) == 1 && msg.String() >= "1" && msg.String() <= "9" {
				d := int(msg.String()[0] - '0')
				if m.hintCell[m.cursor] {
					m.message = "Case d'indice, déjà remplie !"
					return m, nil
				}
				if m.used[d] {
					m.message = fmt.Sprintf("Le %d est déjà posé ! (＞﹏＜)", d)
					return m, nil
				}
				if old := m.grid[m.cursor]; old > 0 {
					m.used[old] = false
				}
				m.grid[m.cursor] = d
				m.used[d] = true
				m.message = fmt.Sprintf("Un petit %d par ici ♡", d)
				if m.full() {
					if m.checkWin() {
						m.won = true
						m.message = "GAGNÉ ! Trop kawaii !! ☆*:.｡.o(≧▽≦)o.｡.:*☆"
						return m, tick()
					}
					m.message = "Hmm, les sommes ne vont pas... (´･_･`)"
				}
			}
		}
	}
	return m, nil
}

// ─── Vue ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.menu {
		return m.center(m.menuView())
	}
	return m.center(m.gameView())
}

// center centre chaque ligne du bloc : d'abord alignées entre elles sur la
// largeur du contenu (titre, palette, grille et messages partagent le même
// axe central), puis tout le bloc est décalé au centre de la fenêtre.
func (m model) center(content string) string {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	maxW := 0
	for _, l := range lines {
		if lw := lipgloss.Width(l); lw > maxW {
			maxW = lw
		}
	}
	for i, l := range lines {
		inner := (maxW - lipgloss.Width(l)) / 2
		if inner < 0 {
			inner = 0
		}
		lines[i] = strings.Repeat(" ", inner) + l
	}
	outer := (m.width - maxW) / 2
	if outer < 0 {
		outer = 0
	}
	side := strings.Repeat(" ", outer)
	for i, l := range lines {
		lines[i] = side + l
	}
	vpad := (m.height - len(lines)) / 2
	if vpad > 0 {
		top := make([]string, vpad)
		lines = append(top, lines...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m model) menuView() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" ♡  F U B U K I  ♡ "))
	b.WriteString("\n\n")
	b.WriteString(menuStyle.Render("Choisis ton niveau :"))
	b.WriteString("\n\n")
	for i, lv := range levels {
		prefix := fmt.Sprintf("  [%d] ", i+1)
		hintWord := "indice"
		if lv.hints > 1 {
			hintWord = "indices"
		}
		line := fmt.Sprintf("%-9s · %d %s", lv.name, lv.hints, hintWord)
		if i == m.levelIdx {
			b.WriteString(selStyle.Render(prefix + line))
		} else {
			b.WriteString(menuStyle.Render(prefix + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓ choisir · entrée jouer · 1/2/3 direct · q quitter"))
	b.WriteString("\n")
	return b.String()
}

func (m model) sumStyleFor(sum, target int, full bool) lipgloss.Style {
	if !full {
		return sumWait
	}
	if sum == target {
		return sumOK
	}
	return sumKO
}

func (m model) gameView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" ♡  F U B U K I  ♡ "))
	b.WriteString("\n")
	b.WriteString(menuStyle.Render("niveau : " + levels[m.levelIdx].name))
	b.WriteString("\n\n")

	// palette : chiffres restants (posés = teinte douce)
	pal := []string{}
	for d := 1; d <= 9; d++ {
		st := palFree
		if m.used[d] {
			st = palUsed
		}
		pal = append(pal, st.Render(fmt.Sprintf("%d", d)))
	}
	b.WriteString(strings.Join(pal, ""))
	b.WriteString("\n\n")

	rows := []string{}
	for i := 0; i < 3; i++ {
		cells := []string{}
		for j := 0; j < 3; j++ {
			idx := 3*i + j
			var style lipgloss.Style
			switch {
			case m.hintCell[idx]:
				style = hintStyle
			case idx == m.cursor && !m.won:
				style = cursorStyle
			default:
				style = cellStyle
			}
			cells = append(cells, style.Render(cellText(m.grid[idx])))
		}
		cells = append(cells, m.sumStyleFor(m.rowSumOf(i), m.rowSum[i], m.rowFull(i)).
			Render(fmt.Sprintf("=%3d", m.rowSum[i])))
		rows = append(rows, strings.Join(cells, ""))
	}
	cols := []string{}
	for j := 0; j < 3; j++ {
		cols = append(cols, m.sumStyleFor(m.colSumOf(j), m.colSum[j], m.colFull(j)).
			Render(fmt.Sprintf("=%3d", m.colSum[j])))
	}
	rows = append(rows, strings.Join(cols, ""))

	b.WriteString(gridBox.Render(strings.Join(rows, "\n")))
	b.WriteString("\n\n")
	b.WriteString(msgStyle.Render(m.message))
	b.WriteString("\n\n")

	if m.won {
		b.WriteString(pawStyle.Render(strings.Repeat("🐾 ♡ ", 1+min(m.paws, 6))) + "\n")
		b.WriteString(helpStyle.Render("[n] rejouer · [m] menu · [q] quitter"))
	} else {
		b.WriteString(helpStyle.Render("↑↓←→ bouger · 1-9 placer · backspace effacer · n rejouer · m menu · q quitter"))
	}
	b.WriteString("\n")
	return b.String()
}

func cellText(v int) string {
	if v < 0 {
		return "·"
	}
	return fmt.Sprintf("%d", v)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Oups, le jeu a glissé : %v\n", err)
		os.Exit(1)
	}
}
