package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultPanelRatio = 0.4
	minPanelRatio     = 0.2
	maxPanelRatio     = 0.8
	defaultDateFormat = "2006-01-02"
	defaultTimeFormat = "15:04"
)

// FocusConfig holds pomodoro/focus timer settings.
type FocusConfig struct {
	WorkDuration       int  `json:"work_duration_min"`
	ShortBreakDuration int  `json:"short_break_duration_min"`
	LongBreakDuration  int  `json:"long_break_duration_min"`
	LongBreakInterval  int  `json:"long_break_interval"`
	DailyGoal          int  `json:"daily_goal"`
	AutoStartBreak     bool `json:"auto_start_break"`
	Sound              bool `json:"sound"`
}

// Config holds user-configurable settings for the application.
type Config struct {
	PanelRatio     float64     `json:"panel_ratio"`
	DateFormat     string      `json:"date_format"`
	TimeFormat     string      `json:"time_format"`
	DateTimeFormat string      `json:"datetime_format"`
	Focus          FocusConfig `json:"focus"`
}

// DefaultConfig returns a Config populated with default values.
func DefaultConfig() Config {
	return Config{
		PanelRatio:     defaultPanelRatio,
		DateFormat:     defaultDateFormat,
		TimeFormat:     defaultTimeFormat,
		DateTimeFormat: defaultDateFormat + " " + defaultTimeFormat,
		Focus: FocusConfig{
			WorkDuration:       25,
			ShortBreakDuration: 5,
			LongBreakDuration:  15,
			LongBreakInterval:  4,
			DailyGoal:          8,
			Sound:              true,
		},
	}
}

// validate clamps PanelRatio to the allowed range and applies defaults for
// zero values.
func (c *Config) validate() {
	if c.PanelRatio == 0 {
		c.PanelRatio = defaultPanelRatio
	}
	if c.PanelRatio < minPanelRatio {
		c.PanelRatio = minPanelRatio
	}
	if c.PanelRatio > maxPanelRatio {
		c.PanelRatio = maxPanelRatio
	}

	c.DateFormat = strings.TrimSpace(c.DateFormat)
	if c.DateFormat == "" {
		c.DateFormat = defaultDateFormat
	}

	c.TimeFormat = strings.TrimSpace(c.TimeFormat)
	if c.TimeFormat == "" {
		c.TimeFormat = defaultTimeFormat
	}

	c.DateTimeFormat = strings.TrimSpace(c.DateTimeFormat)
	if c.DateTimeFormat == "" {
		c.DateTimeFormat = c.DateFormat + " " + c.TimeFormat
	}

	if c.Focus.WorkDuration == 0 {
		c.Focus.WorkDuration = 25
	}
	if c.Focus.ShortBreakDuration == 0 {
		c.Focus.ShortBreakDuration = 5
	}
	if c.Focus.LongBreakDuration == 0 {
		c.Focus.LongBreakDuration = 15
	}
	if c.Focus.LongBreakInterval == 0 {
		c.Focus.LongBreakInterval = 4
	}
	if c.Focus.DailyGoal == 0 {
		c.Focus.DailyGoal = 8
	}

	// Clamp durations.
	if c.Focus.WorkDuration < 1 {
		c.Focus.WorkDuration = 1
	}
	if c.Focus.WorkDuration > 120 {
		c.Focus.WorkDuration = 120
	}
	if c.Focus.ShortBreakDuration < 1 {
		c.Focus.ShortBreakDuration = 1
	}
	if c.Focus.LongBreakDuration < 1 {
		c.Focus.LongBreakDuration = 1
	}
	if c.Focus.LongBreakInterval < 1 {
		c.Focus.LongBreakInterval = 1
	}
	if c.Focus.LongBreakInterval > 10 {
		c.Focus.LongBreakInterval = 10
	}
}

func (c Config) normalizedDateFormat() string {
	dateFormat := strings.TrimSpace(c.DateFormat)
	if dateFormat == "" {
		return defaultDateFormat
	}
	return dateFormat
}

func (c Config) normalizedTimeFormat() string {
	timeFormat := strings.TrimSpace(c.TimeFormat)
	if timeFormat == "" {
		return defaultTimeFormat
	}
	return timeFormat
}

func (c Config) normalizedDateTimeFormat() string {
	dateTimeFormat := strings.TrimSpace(c.DateTimeFormat)
	if dateTimeFormat == "" {
		return c.normalizedDateFormat() + " " + c.normalizedTimeFormat()
	}
	return dateTimeFormat
}

func (c Config) FormatDate(t time.Time) string {
	return t.Format(c.normalizedDateFormat())
}

func (c Config) FormatTime(t time.Time) string {
	return t.Format(c.normalizedTimeFormat())
}

func (c Config) FormatDateTime(t time.Time) string {
	return t.Format(c.normalizedDateTimeFormat())
}

func (c Config) FormatNoteTitle(date, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	weekAgo := today.AddDate(0, 0, -6)

	switch {
	case date.Equal(today):
		return "Today, " + c.FormatDate(date)
	case date.Equal(yesterday):
		return "Yesterday, " + c.FormatDate(date)
	case date.After(weekAgo):
		return fmt.Sprintf("%s, %s", date.Format("Mon"), c.FormatDate(date))
	default:
		return c.FormatDate(date)
	}
}

// Path returns the absolute path to the config file (~/.todo-app/config.json).
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config path: %w", err)
	}
	return filepath.Join(home, ".todo-app", "config.json"), nil
}

// Load reads the configuration from the config file. If the file does not
// exist, it returns DefaultConfig without error.
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("parse config: %w", err)
	}
	cfg.validate()
	return cfg, nil
}

// Save writes the configuration to the config file as formatted JSON. It
// creates the parent directory if necessary.
func Save(cfg Config) error {
	cfg.validate()

	p, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
