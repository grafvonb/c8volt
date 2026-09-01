// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"sync"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
)

// opsDurableMilestoneMinimumElapsed is the shared completion-driven interval
// for durable semantic progress milestones.
const opsDurableMilestoneMinimumElapsed = 10 * time.Second

// opsProgressMilestonePacer rate-limits durable progress snapshots while
// allowing shared progress callbacks to invoke it concurrently.
type opsProgressMilestonePacer struct {
	mu                    sync.Mutex
	minimumElapsed        time.Duration
	now                   func() time.Time
	lastMilestoneAt       time.Time
	lastProgressSignature opsProgressMilestoneSignature
	hasMilestoneSignature bool
}

// opsProgressMilestoneSignature captures the event fields that must advance
// before a paced durable milestone is useful.
type opsProgressMilestoneSignature struct {
	Kind             ops.ProgressEventKind
	Phase            string
	CurrentPage      int
	Seen             int
	Selected         int
	Done             int
	Total            int
	CompletedSamples int
}

// newOpsProgressMilestonePacer creates an event-driven pacer anchored to the
// supplied clock so tests never need real sleeps.
func newOpsProgressMilestonePacer(now func() time.Time) *opsProgressMilestonePacer {
	if now == nil {
		now = time.Now
	}
	startedAt := now()
	return &opsProgressMilestonePacer{
		minimumElapsed:  opsDurableMilestoneMinimumElapsed,
		now:             now,
		lastMilestoneAt: startedAt,
	}
}

// AllowDurableMilestone reports whether the event advanced enough to emit one
// durable line for the current output channel and pacing window.
func (p *opsProgressMilestonePacer) AllowDurableMilestone(event ops.ProgressEvent, channel ops.ProgressChannel) bool {
	if p == nil || !opsProgressDurableMilestoneChannelAllowed(channel) {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	signature, ok := opsProgressMilestoneSignatureForEvent(event)
	if !ok {
		return false
	}
	if p.hasMilestoneSignature && !opsProgressMilestoneSignatureAdvanced(signature, p.lastProgressSignature) {
		return false
	}
	now := p.now()
	if now.Sub(p.lastMilestoneAt) < p.minimumElapsed {
		return false
	}
	p.lastMilestoneAt = now
	p.lastProgressSignature = signature
	p.hasMilestoneSignature = true
	return true
}

// opsProgressMilestoneSignatureAdvanced reports whether a new event represents
// real progress beyond the last durable line.
func opsProgressMilestoneSignatureAdvanced(current opsProgressMilestoneSignature, previous opsProgressMilestoneSignature) bool {
	if current.Kind != previous.Kind || current.Phase != previous.Phase {
		return true
	}
	switch current.Kind {
	case ops.ProgressEventKindPage:
		return current.CurrentPage > previous.CurrentPage || current.Seen > previous.Seen || current.Selected > previous.Selected
	case ops.ProgressEventKindFrozenScope:
		return current.Done > previous.Done
	case ops.ProgressEventKindETA:
		return current.CompletedSamples > previous.CompletedSamples
	default:
		return current != previous
	}
}

// opsProgressDurableMilestoneChannelAllowed keeps paced progress limited to
// default human stderr so machine-oriented modes stay parseable.
func opsProgressDurableMilestoneChannelAllowed(channel ops.ProgressChannel) bool {
	return channel.Mode == ops.ProgressModeHuman && channel.DurableAllowed && channel.StderrAllowed && !channel.StdoutAllowed
}

// opsProgressMilestoneSignatureForEvent extracts comparable progress fields
// from page, frozen-scope, and ETA events.
func opsProgressMilestoneSignatureForEvent(event ops.ProgressEvent) (opsProgressMilestoneSignature, bool) {
	switch {
	case event.Kind == ops.ProgressEventKindPage && event.Page != nil:
		progress := event.Page
		if progress.CurrentPage <= 0 && progress.Seen <= 0 && progress.Selected <= 0 {
			return opsProgressMilestoneSignature{}, false
		}
		return opsProgressMilestoneSignature{
			Kind:        ops.ProgressEventKindPage,
			Phase:       strings.TrimSpace(progress.Phase),
			CurrentPage: progress.CurrentPage,
			Seen:        progress.Seen,
			Selected:    progress.Selected,
		}, true
	case event.Kind == ops.ProgressEventKindFrozenScope && event.FrozenScope != nil:
		progress := event.FrozenScope
		if progress.Done <= 0 {
			return opsProgressMilestoneSignature{}, false
		}
		return opsProgressMilestoneSignature{
			Kind:  ops.ProgressEventKindFrozenScope,
			Phase: strings.TrimSpace(progress.Phase),
			Done:  progress.Done,
			Total: progress.Total,
		}, true
	case event.Kind == ops.ProgressEventKindETA && event.ETA != nil:
		progress := event.ETA
		if progress.CompletedSamples <= 0 {
			return opsProgressMilestoneSignature{}, false
		}
		return opsProgressMilestoneSignature{
			Kind:             ops.ProgressEventKindETA,
			Phase:            strings.TrimSpace(progress.Phase),
			CompletedSamples: progress.CompletedSamples,
			Total:            progress.Total,
		}, true
	default:
		return opsProgressMilestoneSignature{}, false
	}
}
