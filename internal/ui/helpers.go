package ui

import (
	"fmt"
	"time"
)

// cycleCert advances the tri-state ignore_cert: unset → ignore → enforce → unset.
func cycleCert(c *bool) *bool {
	switch {
	case c == nil:
		t := true
		return &t
	case *c:
		f := false
		return &f
	default:
		return nil
	}
}

// certLabel renders the tri-state for the table/form.
func certLabel(c *bool) string {
	switch {
	case c == nil:
		return "ask"
	case *c:
		return "ignore"
	default:
		return "enforce"
	}
}

// relTime formats a unix timestamp as a short relative age. 0 = never used.
func relTime(ts int64) string {
	if ts == 0 {
		return "never"
	}
	d := time.Since(time.Unix(ts, 0))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func sessionStatus(state sessionState) string {
	switch state {
	case sessionStarting:
		return "starting"
	case sessionActive:
		return "active"
	case sessionExited:
		return "exited"
	case sessionFailed:
		return "failed"
	default:
		return "idle"
	}
}
