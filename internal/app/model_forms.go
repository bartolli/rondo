package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/roniel/todo-app/internal/task"
)

// updateFormMsg handles ALL message types for form modes (not just KeyMsg).
func (m *Model) updateFormMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window resize even during form mode
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.resizeComponents()
	}

	// Only intercept Esc from key messages
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		m.mode = modeNormal
		m.form = nil
		m.formData = nil
		return m, nil
	}

	// Pass all messages to the form (cursor blink, timers, keys, etc.)
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		if m.form.State == huh.StateCompleted {
			if m.mode == modeSubtask {
				selected := m.selectedTask()
				title := ""
				if m.formData != nil {
					title = m.formData.Title
				}
				if selected != nil && title != "" {
					if err := m.store.AddSubtask(selected.ID, title); err != nil {
						return m, m.setError(err)
					}
				}
				m.mode = modeNormal
				m.form = nil
				m.formData = nil
				if err := m.reload(); err != nil {
					return m, m.setError(err)
				}
				return m, m.setStatus("Subtask added")
			}
			if m.mode == modeEditSubtask {
				selected := m.selectedTask()
				title := ""
				if m.formData != nil {
					title = m.formData.Title
				}
				if selected != nil && title != "" && m.subtaskIdx >= 0 && m.subtaskIdx < len(selected.Subtasks) {
					st := selected.Subtasks[m.subtaskIdx]
					if err := m.store.UpdateSubtask(st.ID, title); err != nil {
						return m, m.setError(err)
					}
				}
				m.mode = modeNormal
				m.form = nil
				m.formData = nil
				if err := m.reload(); err != nil {
					return m, m.setError(err)
				}
				return m, m.setStatus("Subtask updated")
			}
			cmd := m.submitTaskForm()
			return m, cmd
		}
		if m.form.State == huh.StateAborted {
			m.mode = modeNormal
			m.form = nil
			m.formData = nil
			return m, nil
		}
	}
	return m, cmd
}

func (m *Model) updateExportForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.resizeComponents()
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		m.mode = modeNormal
		m.form = nil
		m.exportFormData = nil
		return m, nil
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		if m.form.State == huh.StateCompleted {
			data := m.exportFormData
			m.mode = modeNormal
			m.form = nil
			m.exportFormData = nil
			return m, m.runExport(data)
		}
		if m.form.State == huh.StateAborted {
			m.mode = modeNormal
			m.form = nil
			m.exportFormData = nil
			return m, nil
		}
	}
	return m, cmd
}

func (m *Model) updateTimeLogForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.resizeComponents()
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		m.mode = modeNormal
		m.form = nil
		m.timeLogFormData = nil
		return m, nil
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		if m.form.State == huh.StateCompleted {
			selected := m.selectedTask()
			if selected != nil && m.timeLogFormData != nil {
				dur, err := task.ParseDuration(m.timeLogFormData.Duration)
				if err == nil {
					if err := m.store.AddTimeLog(selected.ID, dur, m.timeLogFormData.Note); err != nil {
						m.mode = modeNormal
						m.form = nil
						m.timeLogFormData = nil
						return m, m.setError(err)
					}
				}
			}
			m.mode = modeNormal
			m.form = nil
			m.timeLogFormData = nil
			if err := m.reload(); err != nil {
				return m, m.setError(err)
			}
			return m, m.setStatus("Time logged")
		}
		if m.form.State == huh.StateAborted {
			m.mode = modeNormal
			m.form = nil
			m.timeLogFormData = nil
			return m, nil
		}
	}
	return m, cmd
}

func (m *Model) updateNoteForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.resizeComponents()
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		m.mode = modeNormal
		m.form = nil
		m.noteFormData = nil
		return m, nil
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		if m.form.State == huh.StateCompleted {
			wasEdit := m.mode == modeNoteEdit
			selected := m.selectedTask()
			if selected != nil && m.noteFormData != nil {
				body := strings.TrimSpace(m.noteFormData.Body)
				if body != "" {
					if !wasEdit {
						if err := m.store.AddNote(selected.ID, body); err != nil {
							m.mode = modeNormal
							m.form = nil
							m.noteFormData = nil
							return m, m.setError(err)
						}
					} else {
						if m.noteIdx >= 0 && m.noteIdx < len(selected.Notes) {
							n := selected.Notes[m.noteIdx]
							if err := m.store.UpdateNote(n.ID, body); err != nil {
								m.mode = modeNormal
								m.form = nil
								m.noteFormData = nil
								return m, m.setError(err)
							}
						}
					}
				}
			}
			m.mode = modeNormal
			m.form = nil
			m.noteFormData = nil
			if err := m.reload(); err != nil {
				return m, m.setError(err)
			}
			if wasEdit {
				return m, m.setStatus("Note updated")
			}
			return m, m.setStatus("Note added")
		}
		if m.form.State == huh.StateAborted {
			m.mode = modeNormal
			m.form = nil
			m.noteFormData = nil
			return m, nil
		}
	}
	return m, cmd
}

func (m *Model) updateConfirmDeleteNote(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		selected := m.selectedTask()
		if selected != nil && m.noteIdx >= 0 && m.noteIdx < len(selected.Notes) {
			n := selected.Notes[m.noteIdx]
			// Capture for undo.
			taskID := selected.ID
			noteBody := n.Body
			noteCreatedAt := n.CreatedAt
			m.undoAction = &undoAction{
				description: fmt.Sprintf("Undo delete note %q", truncate(noteBody, 30)),
				undo: func() error {
					return m.store.RestoreNote(taskID, noteBody, noteCreatedAt)
				},
			}
			if err := m.store.DeleteNote(n.ID); err != nil {
				m.mode = modeNormal
				return m, m.setError(err)
			}
			if m.noteIdx > 0 && m.noteIdx >= len(selected.Notes)-1 {
				m.noteIdx--
			}
			if err := m.reload(); err != nil {
				m.mode = modeNormal
				return m, m.setError(err)
			}
		}
		m.mode = modeNormal
		return m, m.setStatus("Note deleted")
	case "n", "N", "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (m *Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		selected := m.selectedTask()
		if selected != nil {
			// Capture for undo.
			deletedTask := *selected
			m.undoAction = &undoAction{
				description: fmt.Sprintf("Undo delete %q", deletedTask.Title),
				undo: func() error {
					return m.store.Restore(&deletedTask)
				},
			}
			if err := m.store.Delete(selected.ID); err != nil {
				return m, m.setError(err)
			}
			if err := m.reload(); err != nil {
				m.mode = modeNormal
				return m, m.setError(err)
			}
		}
		m.mode = modeNormal
		return m, m.setStatus("Task deleted")
	case "n", "N", "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) updateConfirmDeleteSubtask(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		selected := m.selectedTask()
		if selected != nil && m.subtaskIdx >= 0 && m.subtaskIdx < len(selected.Subtasks) {
			st := selected.Subtasks[m.subtaskIdx]
			// Capture for undo.
			taskID := selected.ID
			stTitle := st.Title
			stCompleted := st.Completed
			stPosition := st.Position
			m.undoAction = &undoAction{
				description: fmt.Sprintf("Undo delete subtask %q", stTitle),
				undo: func() error {
					return m.store.RestoreSubtask(taskID, stTitle, stCompleted, stPosition)
				},
			}
			if err := m.store.DeleteSubtask(st.ID); err != nil {
				m.mode = modeNormal
				return m, m.setError(err)
			}
			if m.subtaskIdx > 0 && m.subtaskIdx >= len(selected.Subtasks)-1 {
				m.subtaskIdx--
			}
			if err := m.reload(); err != nil {
				m.mode = modeNormal
				return m, m.setError(err)
			}
		}
		m.mode = modeNormal
		return m, m.setStatus("Subtask deleted")
	case "n", "N", "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) updateFocusConfirmCancel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.focusPhase = phaseIdle
		m.focusSession = nil
		m.mode = modeNormal
		return m, m.setStatus("Focus session cancelled")
	case "n", "N", "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) submitTaskForm() tea.Cmd {
	if m.formData == nil {
		return nil
	}

	var dueDate *time.Time
	if m.formData.DueDate != "" {
		if d, err := time.Parse(time.DateOnly, m.formData.DueDate); err == nil {
			dueDate = &d
		}
	}

	var tags []string
	if m.formData.Tags != "" {
		for _, t := range strings.Split(m.formData.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	recurFreq := task.ParseRecurFreq(m.formData.RecurFreq)

	// Parse metadata.
	metadata := make(map[string]string)
	if m.formData.Metadata != "" {
		for _, pair := range strings.Split(m.formData.Metadata, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if k != "" {
					metadata[k] = v
				}
			}
		}
	}

	// Parse blocks.
	var blocksIDs []int64
	if m.formData.Blocks != "" {
		for _, part := range strings.Split(m.formData.Blocks, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err == nil && id > 0 {
				blocksIDs = append(blocksIDs, id)
			}
		}
	}

	var statusCmd tea.Cmd
	if m.mode == modeAdd {
		t := &task.Task{
			Title:         m.formData.Title,
			Description:   m.formData.Description,
			Priority:      m.formData.Priority,
			DueDate:       dueDate,
			Tags:          tags,
			RecurFreq:     recurFreq,
			RecurInterval: 1,
			Metadata:      metadata,
		}
		if err := m.store.Create(t); err != nil {
			statusCmd = m.setError(err)
		} else {
			if recurFreq != task.RecurNone {
				_ = m.store.UpdateRecurrence(t.ID, recurFreq, 1)
			}
			if len(blocksIDs) > 0 {
				// This task blocks the listed tasks.
				for _, blockedID := range blocksIDs {
					_ = m.store.SetBlocker(blockedID, t.ID)
				}
			}
			statusCmd = m.setStatus("Task created")
		}
	} else if m.mode == modeEdit {
		selected := m.selectedTask()
		if selected != nil {
			selected.Title = m.formData.Title
			selected.Description = m.formData.Description
			selected.Priority = m.formData.Priority
			selected.DueDate = dueDate
			selected.Tags = tags
			selected.RecurFreq = recurFreq
			selected.Metadata = metadata
			if recurFreq != task.RecurNone && selected.RecurInterval == 0 {
				selected.RecurInterval = 1
			}
			if err := m.store.Update(selected); err != nil {
				statusCmd = m.setError(err)
			} else {
				_ = m.store.UpdateRecurrence(selected.ID, recurFreq, selected.RecurInterval)
				// Update blocks: clear old, set new.
				// "Blocks" means this task blocks the listed task IDs.
				// First remove old "blocks" relationships where this task was the blocker.
				oldBlocks := selected.BlocksIDs
				for _, bid := range oldBlocks {
					_ = m.store.RemoveBlocker(bid, selected.ID)
				}
				for _, blockedID := range blocksIDs {
					_ = m.store.SetBlocker(blockedID, selected.ID)
				}
				statusCmd = m.setStatus("Task updated")
			}
		}
	}

	m.mode = modeNormal
	m.form = nil
	m.formData = nil
	if err := m.reload(); err != nil {
		return m.setError(err)
	}
	return statusCmd
}
