/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package doctor

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
)

const (
	componentColumnWidth = 14
	statusColumnWidth    = 12
	labelWidth           = 8
)

func printReport(out io.Writer, report *Report) error {
	style := newOutputStyle(out)
	if err := printHeader(out, report, style); err != nil {
		return err
	}
	if err := printComponents(out, report, style); err != nil {
		return err
	}
	if err := printProblems(out, report, style); err != nil {
		return err
	}
	passed, failed, skipped := countChecks(report)
	_, err := fmt.Fprintf(out, "\n%s%s%s, %s, %s\n",
		style.label("Summary"),
		style.separator(),
		style.checkCount(platformstatus.Healthy, passed, "passed"),
		style.checkCount(platformstatus.Unhealthy, failed, "failed"),
		style.checkCount(platformstatus.Skipped, skipped, "skipped"),
	)
	return err
}

func printHeader(out io.Writer, report *Report, style outputStyle) error {
	if _, err := fmt.Fprintln(out, style.title("KubeClipper Doctor")); err != nil {
		return err
	}
	overall := string(report.Status)
	if style.enabled {
		overall = statusMarker(report.Status, true) + " " + overall
	}
	if _, err := fmt.Fprintf(out, "\n%s%s%s\n",
		style.label("Overall"), style.separator(), style.status(report.Status, overall)); err != nil {
		return err
	}
	checkedAt := report.CheckedAt.Local().Format(time.RFC3339)
	duration := style.muted("(" + formatDuration(report.DurationMillis) + ")")
	if _, err := fmt.Fprintf(out, "%s%s%s %s\n", style.label("Checked"), style.separator(), checkedAt, duration); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out)
	return err
}

func printComponents(out io.Writer, report *Report, style outputStyle) error {
	if style.enabled {
		if _, err := fmt.Fprintf(out, "%s  %s  %s\n",
			style.heading(padRight("COMPONENT", componentColumnWidth)),
			style.heading(padRight("STATUS", statusColumnWidth)),
			style.heading("DETAILS")); err != nil {
			return err
		}
	}
	for _, component := range report.Components {
		marker := statusMarker(component.Status, style.enabled)
		name := component.Name
		if style.enabled {
			status := marker + " " + string(component.Status)
			if _, err := fmt.Fprintf(out, "%s  %s  %s\n",
				padRight(name, componentColumnWidth),
				style.status(component.Status, padRight(status, statusColumnWidth)),
				component.Message); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(out, "%-5s %-13s %s\n", marker, name, component.Message); err != nil {
			return err
		}
	}
	return nil
}

func printProblems(out io.Writer, report *Report, style outputStyle) error {
	problem := 0
	for componentIndex := range report.Components {
		component := &report.Components[componentIndex]
		for checkIndex := range component.Checks {
			check := &component.Checks[checkIndex]
			if check.Status == platformstatus.Healthy || check.Status == platformstatus.Skipped {
				continue
			}
			problem++
			if problem == 1 {
				if _, err := fmt.Fprintln(out, "\n"+style.section("Problems")); err != nil {
					return err
				}
			}
			if err := printProblem(out, problem, check, style); err != nil {
				return err
			}
		}
	}
	return nil
}

func printProblem(out io.Writer, number int, check *Check, style outputStyle) error {
	title := fmt.Sprintf("[%d] %s", number, check.Message)
	if _, err := fmt.Fprintf(out, "\n%s\n", style.status(check.Status, title)); err != nil {
		return err
	}
	if err := printDetailLines(out, style, "Evidence:", check.Evidence, false); err != nil {
		return err
	}
	if err := printDetailLines(out, style, "Recent logs:", check.Logs, false); err != nil {
		return err
	}
	return printDetailLines(out, style, "Next steps:", check.Commands, true)
}

func printDetailLines(out io.Writer, style outputStyle, heading string, lines []string, command bool) error {
	if len(lines) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(out, "\n%s\n", style.detailHeading(heading)); err != nil {
		return err
	}
	for _, line := range lines {
		if command {
			line = style.command(line)
		}
		if _, err := fmt.Fprintf(out, "  %s\n", line); err != nil {
			return err
		}
	}
	return nil
}

func countChecks(report *Report) (passed, failed, skipped int) {
	for componentIndex := range report.Components {
		component := &report.Components[componentIndex]
		for checkIndex := range component.Checks {
			check := &component.Checks[checkIndex]
			switch check.Status {
			case platformstatus.Healthy:
				passed++
			case platformstatus.Skipped:
				skipped++
			default:
				failed++
			}
		}
	}
	return passed, failed, skipped
}

func statusMarker(status platformstatus.Status, terminal bool) string {
	if terminal {
		switch status {
		case platformstatus.Healthy:
			return "✓"
		case platformstatus.Degraded:
			return "!"
		case platformstatus.Unhealthy:
			return "✗"
		default:
			return "?"
		}
	}
	switch status {
	case platformstatus.Healthy:
		return "[OK]"
	case platformstatus.Degraded:
		return "[WARN]"
	case platformstatus.Unhealthy:
		return "[FAIL]"
	case platformstatus.Skipped:
		return "[SKIP]"
	default:
		return "[UNKNOWN]"
	}
}

func padRight(value string, width int) string {
	displayWidth := utf8.RuneCountInString(value)
	if displayWidth >= width {
		return value
	}
	return value + strings.Repeat(" ", width-displayWidth)
}

type outputStyle struct {
	enabled bool
}

func newOutputStyle(out io.Writer) outputStyle {
	_, noColor := os.LookupEnv("NO_COLOR")
	return outputStyle{
		enabled: writerIsTerminal(out) && !noColor && os.Getenv("TERM") != "dumb",
	}
}

func (s outputStyle) title(value string) string {
	return s.wrap("1;36", value)
}

func (s outputStyle) section(value string) string {
	return s.wrap("1", value)
}

func (s outputStyle) heading(value string) string {
	return s.wrap("2", value)
}

func (s outputStyle) label(value string) string {
	if !s.enabled {
		return value + ":"
	}
	return s.wrap("1", padRight(value, labelWidth))
}

func (s outputStyle) separator() string {
	if s.enabled {
		return "  "
	}
	return " "
}

func (s outputStyle) detailHeading(value string) string {
	return s.wrap("1", value)
}

func (s outputStyle) muted(value string) string {
	return s.wrap("2", value)
}

func (s outputStyle) command(value string) string {
	return s.wrap("36", value)
}

func (s outputStyle) status(status platformstatus.Status, value string) string {
	code := "35"
	switch status {
	case platformstatus.Healthy:
		code = "1;32"
	case platformstatus.Degraded:
		code = "1;33"
	case platformstatus.Unhealthy:
		code = "1;31"
	case platformstatus.Skipped:
		code = "2"
	}
	return s.wrap(code, value)
}

func (s outputStyle) checkCount(status platformstatus.Status, count int, label string) string {
	value := fmt.Sprintf("%d %s", count, label)
	if count == 0 {
		return s.muted(value)
	}
	return s.status(status, value)
}

func (s outputStyle) wrap(code, value string) string {
	if !s.enabled {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}

func writerIsTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func formatDuration(milliseconds int64) string {
	duration := time.Duration(milliseconds) * time.Millisecond
	if duration < time.Second {
		return duration.String()
	}
	value := fmt.Sprintf("%.1f", duration.Seconds())
	return strings.TrimSuffix(value, ".0") + "s"
}
