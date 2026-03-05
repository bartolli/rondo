package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/roniel/todo-app/internal/config"
	"github.com/roniel/todo-app/internal/focus"
	"github.com/roniel/todo-app/internal/task"
	"github.com/roniel/todo-app/internal/ui"
)

func (m Model) renderStatsOverlay() string {
	if m.stats == nil {
		return ""
	}
	s := m.stats

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.White).Render("Statistics"))
	lines = append(lines, "")

	// Task counts.
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render("Tasks"))
	lines = append(lines, fmt.Sprintf("  Total: %d  Active: %d  Done: %d", s.totalTasks, s.activeTasks, s.doneTasks))
	lines = append(lines, "")

	// Priority breakdown.
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render("Priority"))
	lines = append(lines, "  "+ui.RenderPriorityBreakdown(s.lowCount, s.medCount, s.highCount, s.urgentCount))
	lines = append(lines, "")

	// Tags.
	if len(s.tagCounts) > 0 {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render("Tags"))
		lines = append(lines, "  "+ui.RenderTagCloud(s.tagCounts))
		lines = append(lines, "")
	}

	// Focus sessions.
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render("Focus"))

	// Today's progress toward goal.
	if s.focusGoal > 0 {
		lines = append(lines, fmt.Sprintf("  Today: %d/%d sessions", s.focusToday, s.focusGoal))
		// Progress bar toward daily goal.
		filled := s.focusToday * 30 / s.focusGoal
		if filled > 30 {
			filled = 30
		}
		empty := 30 - filled
		bar := lipgloss.NewStyle().Foreground(ui.Green).Render(strings.Repeat("█", filled))
		bar += lipgloss.NewStyle().Foreground(ui.DimGray).Render(strings.Repeat("░", empty))
		lines = append(lines, "  "+bar)
	} else {
		lines = append(lines, fmt.Sprintf("  Today: %d sessions", s.focusToday))
	}

	// 7-day total and sparkline.
	if s.focusTotalMins > 0 || len(s.focusWeekly) > 0 {
		h := s.focusTotalMins / 60
		mins := s.focusTotalMins % 60
		lines = append(lines, fmt.Sprintf("  7-day total: %dh %dm", h, mins))
		data := focusWeeklyData(s.focusWeekly, 7)
		sparkline := ui.RenderSparkline(data, 7)
		if sparkline != "" {
			lines = append(lines, "  7d: "+sparkline)
		}
	}

	// Streak.
	lines = append(lines, fmt.Sprintf("  Streak: %d days", s.focusStreak))
	lines = append(lines, "")

	// Journal streak.
	if s.journalStreak != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render("Journal"))
		lines = append(lines, "  "+s.journalStreak)
		lines = append(lines, "")
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render("Press Esc or G to close"))

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ui.Cyan).
		Padding(1, 3).
		Width(60).
		Render(content)
}

func (m Model) renderBlockerOverlay() string {
	selected := m.selectedTask()
	if selected == nil {
		return ""
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.White).Render("Task Dependencies"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render(fmt.Sprintf("Task: %s", selected.Title)))
	lines = append(lines, "")

	if len(selected.BlockedByIDs) == 0 && len(selected.BlocksIDs) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render("No dependencies"))
	} else {
		if len(selected.BlockedByIDs) > 0 {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render("Blocked by:"))
			for _, id := range selected.BlockedByIDs {
				blocker, err := m.store.GetByID(id)
				if err != nil {
					lines = append(lines, fmt.Sprintf("  - Task #%d (not found)", id))
					continue
				}
				statusIcon := blocker.Status.Icon()
				style := lipgloss.NewStyle().Foreground(ui.White)
				if blocker.Status == task.Done {
					style = style.Foreground(ui.Green)
				}
				lines = append(lines, style.Render(fmt.Sprintf("  %s %s", statusIcon, blocker.Title)))
			}
		}
		if len(selected.BlocksIDs) > 0 {
			if len(selected.BlockedByIDs) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render("Blocks:"))
			for _, id := range selected.BlocksIDs {
				blocked, err := m.store.GetByID(id)
				if err != nil {
					lines = append(lines, fmt.Sprintf("  - Task #%d (not found)", id))
					continue
				}
				statusIcon := blocked.Status.Icon()
				style := lipgloss.NewStyle().Foreground(ui.White)
				if blocked.Status == task.Done {
					style = style.Foreground(ui.Green)
				}
				lines = append(lines, style.Render(fmt.Sprintf("  %s %s", statusIcon, blocked.Title)))
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render("Press Esc to close"))

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ui.Cyan).
		Padding(1, 2).
		Width(50).
		Render(content)
}

func (m *Model) computeStats() {
	s := &statsData{
		totalTasks: len(m.tasks),
		tagCounts:  make(map[string]int),
	}
	for _, t := range m.tasks {
		switch t.Status {
		case task.Done:
			s.doneTasks++
		default:
			s.activeTasks++
		}
		switch t.Priority {
		case task.Low:
			s.lowCount++
		case task.Medium:
			s.medCount++
		case task.High:
			s.highCount++
		case task.Urgent:
			s.urgentCount++
		}
		for _, tag := range t.Tags {
			s.tagCounts[tag]++
		}
	}
	if m.focusStore != nil {
		s.focusToday, _ = m.focusStore.TodayCount()
		s.focusGoal = m.cfg.Focus.DailyGoal
		s.focusStreak, _ = m.focusStore.Streak()
		s.focusWeekly, _ = m.focusStore.WeeklySummary()
		s.focusTotalMins, _ = m.focusStore.TotalMinutesFocused(7)
	}
	if m.journalStore != nil {
		completions := make(map[string]int)
		for _, n := range m.notes {
			if len(n.Entries) > 0 {
				completions[n.Date.Format(time.DateOnly)] = len(n.Entries)
			}
		}
		s.journalStreak = ui.RenderJournalStreak(completions, 30)
	}
	m.stats = s
}

// focusWeeklyData converts the weekly summary map into an ordered slice for the last N days.
func focusWeeklyData(weekly map[string]int, days int) []int {
	data := make([]int, days)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 0; i < days; i++ {
		day := today.AddDate(0, 0, -(days - 1 - i))
		key := day.Format(time.DateOnly)
		data[i] = weekly[key]
	}
	return data
}

// renderPanel draws a bordered panel with the title embedded in the top border (lazygit-style).
func renderPanel(content, title string, width, height int, focused bool) string {
	if width < 4 || height < 3 {
		return content
	}

	borderColor := ui.Gray
	if focused {
		borderColor = ui.Cyan
	}

	bc := lipgloss.NewStyle().Foreground(borderColor)
	tc := lipgloss.NewStyle().Foreground(borderColor).Bold(focused)

	border := lipgloss.RoundedBorder()
	innerWidth := width - 2

	// Pad and size the content area.
	padded := lipgloss.NewStyle().
		Width(innerWidth).
		Height(height - 2).
		PaddingLeft(1).
		PaddingRight(1).
		Render(content)

	// Top border with title: ╭─ 1 Tasks ──────────────╮
	titleRendered := tc.Render(title)
	titleVisualWidth := lipgloss.Width(titleRendered)
	fillWidth := innerWidth - titleVisualWidth - 3 // "─ " before + " " after
	if fillWidth < 0 {
		fillWidth = 0
	}
	topLine := bc.Render(border.TopLeft+border.Top+" ") +
		titleRendered +
		bc.Render(" "+strings.Repeat(border.Top, fillWidth)+border.TopRight)

	// Bottom border: ╰──────────────╯
	bottomLine := bc.Render(border.BottomLeft + strings.Repeat(border.Bottom, innerWidth) + border.BottomRight)

	// Add side borders to each content line.
	lines := strings.Split(padded, "\n")
	result := make([]string, 0, len(lines)+2)
	result = append(result, topLine)
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth > innerWidth {
			line = lipgloss.NewStyle().MaxWidth(innerWidth).Render(line)
			lineWidth = lipgloss.Width(line)
		}
		if lineWidth < innerWidth {
			line += strings.Repeat(" ", innerWidth-lineWidth)
		}
		result = append(result, bc.Render(border.Left)+line+bc.Render(border.Right))
	}
	result = append(result, bottomLine)

	return strings.Join(result, "\n")
}

func (m Model) renderHelpOverlay() string {
	helpLines := []struct{ key, desc string }{
		{"", "Navigation"},
		{"1 / 2", "Focus list / detail panel"},
		{"j/k", "Navigate items"},
		{"Tab", "Switch tab"},
		{"< / >", "Resize panels"},
		{"Esc", "Back to list / clear filter"},
		{"", ""},
		{"", "Tasks (1: list, 2: detail)"},
		{"a / e / d", "Add / edit / delete"},
		{"s", "Cycle status / toggle subtask"},
		{"/", "Search / filter"},
		{"F1/F2/F3", "Sort date / due / priority"},
		{"F4", "Tag filter bar"},
		{"l", "Log time (detail)"},
		{"n", "Notes (detail)"},
		{"b", "View blockers (detail)"},
		{"", ""},
		{"", "Tools"},
		{"p", "Focus timer"},
		{"P", "Focus settings"},
		{"X", "Export"},
		{"G", "Statistics"},
		{"Ctrl+Z", "Undo"},
		{"?", "This help"},
		{"q", "Quit"},
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.White).Render("Keyboard Shortcuts"))
	lines = append(lines, "")
	for _, h := range helpLines {
		if h.key == "" && h.desc == "" {
			lines = append(lines, "")
			continue
		}
		if h.key == "" {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render(h.desc))
			continue
		}
		k := lipgloss.NewStyle().Foreground(ui.Cyan).Width(16).Render(h.key)
		d := lipgloss.NewStyle().Foreground(ui.Gray).Render(h.desc)
		lines = append(lines, k+d)
	}
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render("Press Esc or ? to close"))

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ui.Cyan).
		Padding(1, 3).
		Width(50).
		Render(content)
}

// renderFocusSessionEndOverlay renders the overlay shown when a work session completes.
func (m Model) renderFocusSessionEndOverlay() string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Green).Render("🍅 Work Session Complete!"))
	lines = append(lines, "")

	// Cycle progress indicator.
	cycle := m.cycleIndicator()
	if cycle != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render("Progress: ")+
			lipgloss.NewStyle().Foreground(ui.Green).Render(cycle))
		lines = append(lines, "")
	}

	// What break comes next (based on updated cyclePos).
	var breakType string
	var breakMins int
	if m.focusCyclePos == 0 {
		breakType = "Long Break"
		breakMins = m.cfg.Focus.LongBreakDuration
	} else {
		breakType = "Short Break"
		breakMins = m.cfg.Focus.ShortBreakDuration
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.White).Render(
		fmt.Sprintf("Next: %s (%d min)", breakType, breakMins)))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render(
		"[Enter] Start break  [s] Skip  [Esc] Dismiss"))

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ui.Green).
		Padding(1, 3).
		Width(50).
		Render(content)
}

// renderFocusBreakEndOverlay renders the overlay shown when a break session completes.
func (m Model) renderFocusBreakEndOverlay() string {
	var lines []string

	// Determine the break type that just ended.
	emoji := "☕"
	breakKind := "Short Break"
	if m.focusSession != nil && m.focusSession.Kind == focus.KindLongBreak {
		emoji = "🌿"
		breakKind = "Long Break"
	}

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ui.Cyan).Render(
		fmt.Sprintf("%s %s Complete!", emoji, breakKind)))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.White).Render(
		fmt.Sprintf("Ready for work session (%d min)", m.cfg.Focus.WorkDuration)))

	// Show cycle progress.
	cycle := m.cycleIndicator()
	if cycle != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render("Cycle: ")+
			lipgloss.NewStyle().Foreground(ui.Cyan).Render(cycle))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.Gray).Render(
		"[Enter] Start work  [s] Skip  [Esc] Dismiss"))

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ui.Cyan).
		Padding(1, 3).
		Width(50).
		Render(content)
}

// updateFocusSettingsForm handles all message types while the focus settings form is active.
func (m *Model) updateFocusSettingsForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.resizeComponents()
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		m.mode = modeNormal
		m.form = nil
		m.focusSettingsFormData = nil
		return m, nil
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		if m.form.State == huh.StateCompleted {
			if m.focusSettingsFormData != nil {
				m.applyFocusSettings()
			}
			m.mode = modeNormal
			m.form = nil
			m.focusSettingsFormData = nil
			return m, m.setStatus("Focus settings saved")
		}
		if m.form.State == huh.StateAborted {
			m.mode = modeNormal
			m.form = nil
			m.focusSettingsFormData = nil
			return m, nil
		}
	}
	return m, cmd
}

// applyFocusSettings parses the form data strings and saves the updated config.
func (m *Model) applyFocusSettings() {
	data := m.focusSettingsFormData
	if v, err := strconv.Atoi(data.WorkDuration); err == nil && v > 0 {
		m.cfg.Focus.WorkDuration = v
	}
	if v, err := strconv.Atoi(data.ShortBreakDuration); err == nil && v > 0 {
		m.cfg.Focus.ShortBreakDuration = v
	}
	if v, err := strconv.Atoi(data.LongBreakDuration); err == nil && v > 0 {
		m.cfg.Focus.LongBreakDuration = v
	}
	if v, err := strconv.Atoi(data.SessionsPerSet); err == nil && v > 0 {
		m.cfg.Focus.LongBreakInterval = v
	}
	if v, err := strconv.Atoi(data.DailyGoal); err == nil && v > 0 {
		m.cfg.Focus.DailyGoal = v
	}
	m.cfg.Focus.AutoStartBreak = data.AutoStartBreaks
	m.cfg.Focus.Sound = data.Sound
	_ = config.Save(m.cfg)
}
