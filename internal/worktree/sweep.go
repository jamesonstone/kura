package worktree

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type sweepRunError struct{ count int }

func (err sweepRunError) Error() string {
	return fmt.Sprintf("sweep completed with %d failure(s)", err.count)
}

func (a *App) sweep(ctx context.Context, cwd string, args []string) error {
	if len(args) != 0 && args[0] == "config" {
		return a.runSweepConfigCommand(ctx, args[1:])
	}
	options, err := parseSweepOptions(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		if err := a.offerFirstSweepConfig(); err != nil {
			return err
		}
	}
	config, err := a.loadSweepConfig(options)
	if err != nil {
		return err
	}
	report := a.buildSweepReport(ctx, cwd, config, options)
	failuresBeforeAction := len(report.Failures)
	if options.Explain != "" {
		err = writeSweepExplanation(a.out, report, options.Explain, sweepUseColor(a, options))
	} else if options.Auto {
		if err = persistSweepReview(config.StateRoot, report); err == nil {
			err = a.applySweepCandidates(ctx, cwd, config, options, &report, automaticSweepCandidates(report.Candidates))
		}
	} else if options.Interactive || a.isTerminal() && !options.JSON && !options.DryRun {
		err = a.runSweepTerminal(ctx, cwd, config, options, &report)
	}
	if err != nil && len(report.Failures) == failuresBeforeAction {
		report.addFailure("interaction-or-apply", "", "", err)
	}
	finalizeSweepResult(&report)
	persistErr := persistSweepReport(config.StateRoot, report)
	if persistErr != nil {
		report.addFailure("persist-report", "", config.StateRoot, persistErr)
		finalizeSweepResult(&report)
	}
	var outputErr error
	if options.Explain == "" {
		if options.JSON {
			outputErr = writeSweepJSON(a.out, sweepOutputReport(report, options))
		} else if options.Auto || !options.Interactive && (!a.isTerminal() || options.DryRun) {
			outputErr = writeSweepHuman(a.out, report, options, sweepUseColor(a, options))
		}
	}
	if outputErr != nil {
		outputErr = fmt.Errorf("write sweep output: %w", outputErr)
	}
	var runErr error
	if len(report.Failures) != 0 {
		runErr = sweepRunError{count: len(report.Failures)}
	}
	return errors.Join(runErr, outputErr)
}

func sweepOutputReport(report SweepReport, options SweepOptions) SweepReport {
	if options.Only == "" {
		return report
	}
	projected := report
	projected.Candidates = make([]SweepCandidate, 0)
	for _, candidate := range report.Candidates {
		if candidate.State == options.Only {
			projected.Candidates = append(projected.Candidates, candidate)
		}
	}
	return projected
}

func (a *App) buildSweepReport(
	ctx context.Context,
	cwd string,
	config SweepConfig,
	options SweepOptions,
) SweepReport {
	now := time.Now().UTC()
	a.populateSweepProcessSnapshot(ctx, &config)
	report := SweepReport{
		SchemaVersion: sweepReportSchemaVersion,
		RunID:         now.Format("20060102T150405.000000000Z"),
		GeneratedAt:   now.Format(time.RFC3339Nano),
		ConfigPath:    config.ConfigPath,
		Roots:         append([]string{}, config.Roots...),
		ProjectRoots:  append([]string{}, config.ProjectRoots...),
		Candidates:    make([]SweepCandidate, 0),
		Failures:      make([]SweepFailure, 0),
		Actions:       make([]SweepAction, 0),
	}
	repositories, failures := a.discoverSweepRepositories(ctx, config)
	report.Failures = append(report.Failures, failures...)
	a.classifySweepRepositories(ctx, cwd, config, repositories, &report)
	if config.Sizes {
		populateSweepSizes(ctx, report.Candidates, config.SizeJobs)
	}
	for index := range report.Candidates {
		report.Candidates[index].Snapshot = sweepCandidateSnapshot(report.Candidates[index])
	}
	sortSweepCandidates(report.Candidates, options.Sort)
	finalizeSweepResult(&report)
	return report
}

func finalizeSweepResult(report *SweepReport) {
	switch {
	case len(report.Failures) != 0 && len(report.Actions) != 0:
		report.Result = "partial"
	case len(report.Failures) != 0:
		report.Result = "failed"
	case len(report.Actions) != 0:
		report.Result = "ok"
	default:
		report.Result = "report"
	}
}

func automaticSweepCandidates(candidates []SweepCandidate) []SweepCandidate {
	selected := make([]SweepCandidate, 0)
	for _, candidate := range candidates {
		if candidate.AutoRemovable && (candidate.State == SweepRemoveReady || candidate.State == SweepStaleMetadata) {
			selected = append(selected, candidate)
		}
	}
	return selected
}
