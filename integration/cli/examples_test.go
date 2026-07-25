// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type exampleSourceKind string

const (
	exampleSourceHelp exampleSourceKind = "help"
	exampleSourceDocs exampleSourceKind = "docs"
)

type cliExample struct {
	SourceKind  exampleSourceKind `json:"sourceKind"`
	SourcePath  string            `json:"sourcePath"`
	SourceLine  int               `json:"sourceLine"`
	CommandPath string            `json:"commandPath"`
	Raw         string            `json:"raw"`
	SourceText  string            `json:"-"`
}

type exampleSeedData struct {
	Profile              integrationProfile
	BpmnProcessID        string
	ProcessInstanceKeys  []string
	ProcessDefinitionKey string
	ResourceID           string
	EmbeddedFixturePath  string
}

type exampleValidationRecord struct {
	SourceKind             exampleSourceKind `json:"sourceKind"`
	SourcePath             string            `json:"sourcePath"`
	SourceLine             int               `json:"sourceLine"`
	CommandPath            string            `json:"commandPath"`
	Raw                    string            `json:"raw"`
	Normalized             string            `json:"normalized"`
	Arguments              []string          `json:"arguments,omitempty"`
	Substitutions          []string          `json:"substitutions,omitempty"`
	Destructive            bool              `json:"destructive"`
	WarningPresent         bool              `json:"warningPresent"`
	DestructiveWarning     string            `json:"destructiveWarning,omitempty"`
	ExecutionMode          string            `json:"executionMode"`
	Outcome                string            `json:"outcome"`
	FailureClass           string            `json:"failureClass,omitempty"`
	FailureReason          string            `json:"failureReason,omitempty"`
	StdoutPath             string            `json:"stdoutPath,omitempty"`
	StderrPath             string            `json:"stderrPath,omitempty"`
	ExitCode               int               `json:"exitCode,omitempty"`
	Profile                string            `json:"profile,omitempty"`
	CamundaVersion         string            `json:"camundaVersion,omitempty"`
	UnresolvedPlaceholders []string          `json:"unresolvedPlaceholders,omitempty"`
}

type examplesReport struct {
	Examples []exampleValidationRecord `json:"examples"`
	Summary  examplesSummary           `json:"summary"`
}

type examplesSummary struct {
	Total               int `json:"total"`
	HelpSources         int `json:"helpSources"`
	DocSources          int `json:"docSources"`
	Executed            int `json:"executed"`
	Substituted         int `json:"substituted"`
	Destructive         int `json:"destructive"`
	DestructiveWarnings int `json:"destructiveWarnings"`
	Skipped             int `json:"skipped"`
	Blocked             int `json:"blocked"`
	Failed              int `json:"failed"`
}

func TestHelpExampleExtraction(t *testing.T) {
	output := `Usage:
  c8volt get process-instance [flags]

Examples:
  ./c8volt get pi --key <process-instance-key>
  ./c8volt get pi --state active --limit 5

Flags:
  -h, --help   help for process-instance
`
	examples := extractHelpExamples("help:get process-instance", output)
	if len(examples) != 2 {
		t.Fatalf("help examples = %d, want 2", len(examples))
	}
	if examples[0].SourceLine != 5 {
		t.Fatalf("first source line = %d, want 5", examples[0].SourceLine)
	}
	if examples[0].Raw != "./c8volt get pi --key <process-instance-key>" {
		t.Fatalf("first example = %q", examples[0].Raw)
	}
}

func TestGeneratedDocsExampleExtraction(t *testing.T) {
	examples, err := extractGeneratedDocExamples(filepath.Join(suite.repoRoot, "docs", "cli"))
	if err != nil {
		t.Fatalf("extract generated doc examples: %v", err)
	}
	if len(examples) == 0 {
		t.Fatal("expected generated CLI docs examples")
	}
	assertHasExampleFromSource(t, examples, "c8volt_get_process-instance.md")
}

func TestPlaceholderSubstitution(t *testing.T) {
	seed := exampleSeedData{
		BpmnProcessID:        "C88_SimpleUserTask",
		ProcessInstanceKeys:  []string{"2251799813685250", "2251799813685251"},
		ProcessDefinitionKey: "2251799813685249",
		ResourceID:           "processdefinitions/C88_SimpleUserTask.bpmn",
		EmbeddedFixturePath:  "processdefinitions/C88_SimpleUserTask.bpmn",
	}
	normalized, substitutions := substituteExamplePlaceholders("./c8volt get pi --key <process-instance-key> --key <another-process-instance-key> --bpmn-process-id <bpmn-process-id>", seed)
	if normalized != "./c8volt get pi --key 2251799813685250 --key 2251799813685251 --bpmn-process-id C88_SimpleUserTask" {
		t.Fatalf("normalized example = %q", normalized)
	}
	if len(substitutions) != 3 {
		t.Fatalf("substitutions = %v, want 3", substitutions)
	}
}

func TestDestructiveWarningDetection(t *testing.T) {
	source := "Use --dry-run to preview selected instances without cancelling. Use --auto-confirm for unattended destructive runs."
	if !hasDestructiveWarning(source) {
		t.Fatal("expected destructive warning context")
	}
	if hasDestructiveWarning("List process instances by key.") {
		t.Fatal("unexpected destructive warning context")
	}
}

func TestExamples(t *testing.T) {
	helpExamples := extractLiveHelpExamples(t)
	docExamples, err := extractGeneratedDocExamples(filepath.Join(suite.repoRoot, "docs", "cli"))
	if err != nil {
		t.Fatalf("extract generated doc examples: %v", err)
	}
	if len(helpExamples) == 0 {
		t.Fatal("expected command help examples")
	}
	if len(docExamples) == 0 {
		t.Fatal("expected generated docs examples")
	}

	seed, haveSeed := prepareExampleSeedData(t)
	examples := append(helpExamples, docExamples...)
	records := make([]exampleValidationRecord, 0, len(examples))
	var failures []string
	for _, example := range examples {
		record := validateExample(t, example, seed, haveSeed)
		records = append(records, record)
		if record.Outcome == "fail" {
			failures = append(failures, fmt.Sprintf("%s:%d %s: %s", record.SourcePath, record.SourceLine, record.Raw, record.FailureReason))
		}
		if (record.Outcome == "blocked" || record.Outcome == "skipped") && !exampleNonExecutionAllowed(record) {
			failures = append(failures, fmt.Sprintf("%s:%d %s: example was %s without an allowlisted reason: %s", record.SourcePath, record.SourceLine, record.Raw, record.Outcome, record.FailureReason))
		}
	}

	report := examplesReport{
		Examples: records,
		Summary:  summarizeExampleRecords(records),
	}
	writeJSON(t, "examples.json", report)

	if report.Summary.Substituted == 0 {
		t.Fatal("expected at least one example placeholder substitution")
	}
	if report.Summary.Destructive == 0 {
		t.Fatal("expected destructive example coverage")
	}
	if report.Summary.DestructiveWarnings != report.Summary.Destructive {
		t.Fatalf("destructive warning records = %d, want %d", report.Summary.DestructiveWarnings, report.Summary.Destructive)
	}
	if report.Summary.Executed == 0 {
		t.Fatal("expected at least one executable example")
	}
	if haveSeed && !hasDisposableTargetExecution(records) {
		t.Fatal("expected at least one disposable-target example execution")
	}
	if len(failures) > 0 {
		t.Fatalf("example validation failures:\n%s", strings.Join(failures, "\n"))
	}
}

func extractLiveHelpExamples(t *testing.T) []cliExample {
	t.Helper()
	paths := sortedManifestPaths()
	var examples []cliExample
	for _, path := range paths {
		result := runC8Volt(t, "examples-help-"+path, append(strings.Fields(path), "--help")...)
		if result.Err != nil {
			t.Fatalf("%s help failed: %v\nstderr:\n%s", path, result.Err, result.Stderr)
		}
		sourcePath := "help:" + path
		for _, example := range extractHelpExamples(sourcePath, result.Stdout) {
			example.CommandPath = path
			example.SourceText = result.Stdout
			examples = append(examples, example)
		}
	}
	return examples
}

func extractHelpExamples(sourcePath string, output string) []cliExample {
	lines := strings.Split(output, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "Examples:" {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return nil
	}

	var examples []cliExample
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			if len(examples) > 0 {
				break
			}
			continue
		}
		if strings.HasSuffix(trimmed, ":") {
			break
		}
		if isExampleCommand(trimmed) {
			examples = append(examples, cliExample{
				SourceKind: exampleSourceHelp,
				SourcePath: sourcePath,
				SourceLine: i + 1,
				Raw:        trimmed,
			})
		}
	}
	return examples
}

func extractGeneratedDocExamples(docsDir string) ([]cliExample, error) {
	paths, err := filepath.Glob(filepath.Join(docsDir, "c8volt*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	var examples []cliExample
	for _, path := range paths {
		if filepath.Base(path) == "index.md" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		source := string(data)
		lines := strings.Split(source, "\n")
		inExamples := false
		inFence := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(trimmed, "### ") && trimmed != "### Examples":
				inExamples = false
				inFence = false
			case trimmed == "### Examples":
				inExamples = true
			case inExamples && strings.HasPrefix(trimmed, "```"):
				inFence = !inFence
			case inExamples && inFence && isExampleCommand(trimmed):
				examples = append(examples, cliExample{
					SourceKind:  exampleSourceDocs,
					SourcePath:  path,
					SourceLine:  i + 1,
					CommandPath: commandPathFromDocPath(path),
					Raw:         trimmed,
					SourceText:  source,
				})
			}
		}
	}
	return examples, nil
}

func validateExample(t *testing.T, example cliExample, seed exampleSeedData, haveSeed bool) exampleValidationRecord {
	t.Helper()
	normalized, substitutions := substituteExamplePlaceholders(example.Raw, seed)
	args, parseErr := exampleCommandArgs(normalized)
	commandPath := example.CommandPath
	if resolved := resolveExampleCommandPath(args); resolved != "" {
		commandPath = resolved
	}
	destructive := exampleIsDestructive(commandPath, normalized)
	warningPresent := !destructive || hasDestructiveWarning(exampleSourceText(t, example))

	record := exampleValidationRecord{
		SourceKind:         example.SourceKind,
		SourcePath:         displayExampleSourcePath(example.SourcePath),
		SourceLine:         example.SourceLine,
		CommandPath:        commandPath,
		Raw:                example.Raw,
		Normalized:         normalized,
		Substitutions:      substitutions,
		Destructive:        destructive,
		WarningPresent:     warningPresent,
		DestructiveWarning: destructiveWarningSource(exampleSourceText(t, example)),
		Outcome:            "skipped",
		ExecutionMode:      "not-executed",
	}
	if parseErr != nil {
		record.Outcome = "blocked"
		record.FailureClass = "harness_setup"
		record.FailureReason = parseErr.Error()
		return record
	}
	record.Arguments = args
	record.UnresolvedPlaceholders = unresolvedPlaceholders(normalized)
	if len(record.UnresolvedPlaceholders) > 0 {
		record.Outcome = "blocked"
		record.FailureClass = "harness_setup"
		record.FailureReason = "unresolved placeholders: " + strings.Join(record.UnresolvedPlaceholders, ", ")
		return record
	}
	if hasExplicitConfigArg(args) {
		record.Outcome = "blocked"
		record.FailureClass = "harness_setup"
		record.FailureReason = "example passes --config, which this suite intentionally rejects"
		return record
	}
	if hasExplicitProfileArg(args) {
		record.Outcome = "blocked"
		record.FailureClass = "harness_setup"
		record.FailureReason = "example selects a documentation profile that is not the suite-selected disposable profile"
		return record
	}
	if hasHardCodedDemoSelector(normalized) {
		record.Outcome = "blocked"
		record.FailureClass = "harness_setup"
		record.FailureReason = "example uses hard-coded documentation selectors that are not suite-owned"
		return record
	}
	if !warningPresent {
		record.Outcome = "fail"
		record.FailureClass = "product"
		record.FailureReason = "destructive example is not marked with warning context"
		return record
	}

	mode, runnable := exampleExecutionMode(args, commandPath, destructive, haveSeed)
	record.ExecutionMode = mode
	if !runnable {
		record.FailureReason = mode
		return record
	}

	var result commandResult
	if mode == "disposable-profile" || mode == "disposable-dry-run" {
		result = runC8VoltForProfile(t, seed.Profile.Name, "example-"+sanitizeEvidenceName(record.SourcePath)+"-"+fmt.Sprint(record.SourceLine), args...)
		record.Profile = seed.Profile.Name
		record.CamundaVersion = seed.Profile.ExpectedVersion
	} else {
		result = runC8Volt(t, "example-"+sanitizeEvidenceName(record.SourcePath)+"-"+fmt.Sprint(record.SourceLine), args...)
	}
	record.StdoutPath = result.StdoutPath
	record.StderrPath = result.StderrPath
	record.ExitCode = result.ExitCode
	if result.Err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		record.FailureReason = strings.TrimSpace(result.Stderr)
		if record.FailureReason == "" {
			record.FailureReason = result.Err.Error()
		}
		return record
	}
	record.Outcome = "pass"
	return record
}

func prepareExampleSeedData(t *testing.T) (exampleSeedData, bool) {
	t.Helper()
	profiles, err := selectedProfilesFromDefaultConfig()
	if err != nil {
		t.Fatalf("select integration profiles: %v", err)
	}
	if len(profiles) == 0 {
		return exampleSeedData{}, false
	}
	profile := profiles[0]
	if _, err := checkProfileReadiness(t, profile); err != nil {
		t.Fatalf("profile readiness failed before example validation: %v", err)
	}
	data, records, cleanup, err := seedProfileData(t, profile)
	writeDataEvidence(t, "examples-seeded-data.json", seededDataReport{
		Marker:   suite.marker,
		Profiles: []seededProfileData{data},
		Records:  records,
		Cleanup:  cleanup,
	})
	if err != nil {
		t.Fatalf("seed example data for profile %q: %v", profile.Name, err)
	}
	seed := exampleSeedData{
		Profile:             profile,
		BpmnProcessID:       data.BpmnProcessID,
		ProcessInstanceKeys: data.ProcessInstanceKeys,
		EmbeddedFixturePath: data.FixturePath,
	}
	if len(data.ProcessDefinitionKeys) > 0 {
		seed.ProcessDefinitionKey = data.ProcessDefinitionKeys[0]
	}
	if resourceID := firstString(data.ResourceIDs); isNumericString(resourceID) {
		seed.ResourceID = resourceID
	}
	return seed, true
}

func substituteExamplePlaceholders(raw string, seed exampleSeedData) (string, []string) {
	replacements := map[string]string{
		"<bpmn-process-id>":              seed.BpmnProcessID,
		"<long-running-bpmn-process-id>": seed.BpmnProcessID,
		"<process-instance-key>":         firstString(seed.ProcessInstanceKeys),
		"<another-process-instance-key>": stringAtOrFirst(seed.ProcessInstanceKeys, 1),
		"<process-instance-key-a>":       firstString(seed.ProcessInstanceKeys),
		"<process-instance-key-b>":       stringAtOrFirst(seed.ProcessInstanceKeys, 1),
		"<process-definition-key>":       seed.ProcessDefinitionKey,
		"<resource-id>":                  seed.ResourceID,
	}
	if seed.EmbeddedFixturePath != "" {
		replacements["processdefinitions/<embedded-process>.bpmn"] = seed.EmbeddedFixturePath
	}

	normalized := raw
	var substitutions []string
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := replacements[key]
		if value == "" || !strings.Contains(normalized, key) {
			continue
		}
		normalized = strings.ReplaceAll(normalized, key, value)
		substitutions = append(substitutions, key+"="+value)
	}
	return normalized, substitutions
}

func exampleCommandArgs(command string) ([]string, error) {
	if strings.Contains(command, "|") {
		return nil, fmt.Errorf("pipeline examples are recorded but not subprocess-executed by this validator")
	}
	fields, err := shellFields(command)
	if err != nil {
		return nil, err
	}
	for len(fields) > 0 && fields[0] == "env" {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty example command")
	}
	if fields[0] == "printf" {
		return nil, fmt.Errorf("stdin-producing shell examples are recorded but not subprocess-executed by this validator")
	}
	if !isC8VoltToken(fields[0]) {
		return nil, fmt.Errorf("example does not start with c8volt: %s", command)
	}
	return append([]string(nil), fields[1:]...), nil
}

func shellFields(command string) ([]string, error) {
	var fields []string
	var b strings.Builder
	var quote rune
	escaped := false
	for _, r := range command {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t':
			if b.Len() > 0 {
				fields = append(fields, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in example: %s", command)
	}
	if b.Len() > 0 {
		fields = append(fields, b.String())
	}
	return fields, nil
}

func exampleExecutionMode(args []string, commandPath string, destructive bool, haveSeed bool) (string, bool) {
	if len(args) == 0 {
		return "empty command", false
	}
	if isLocalOnlyExample(commandPath, args) {
		return "local", true
	}
	if !haveSeed && requiresDisposableProfile(commandPath) {
		return "requires selected disposable profile", false
	}
	if destructive {
		if !haveSeed {
			return "requires selected disposable profile", false
		}
		if hasArg(args, "--dry-run") || hasArg(args, "--help") {
			return "disposable-dry-run", true
		}
		return "mutating example requires a dedicated disposable scenario before execution", false
	}
	if requiresDisposableProfile(commandPath) {
		if !haveSeed {
			return "requires selected disposable profile", false
		}
		return "disposable-profile", true
	}
	return "local", true
}

func isLocalOnlyExample(commandPath string, args []string) bool {
	if hasArg(args, "--help") {
		return true
	}
	switch commandPath {
	case "version", "capabilities", "config template", "embed list":
		return true
	default:
		return false
	}
}

func requiresDisposableProfile(commandPath string) bool {
	if commandPath == "" {
		return true
	}
	switch commandPath {
	case "version", "capabilities", "config template", "embed list":
		return false
	default:
		return true
	}
}

func resolveExampleCommandPath(args []string) string {
	candidates := exampleCommandCandidates()
	for len(args) > 0 && isRootFlag(args[0]) {
		if rootFlagConsumesValue(args[0]) && len(args) > 1 {
			args = args[2:]
		} else {
			args = args[1:]
		}
	}
	for length := len(args); length > 0; length-- {
		key := strings.Join(args[:length], " ")
		if path := candidates[key]; path != "" {
			return path
		}
	}
	return ""
}

func exampleCommandCandidates() map[string]string {
	candidates := make(map[string]string, len(commandCoverageManifest)*2)
	for path, entry := range commandCoverageManifest {
		words := strings.Fields(path)
		candidates[strings.Join(words, " ")] = path
		for _, alias := range entry.Aliases {
			aliasWords := append([]string(nil), words...)
			if len(aliasWords) == 0 {
				continue
			}
			aliasWords[len(aliasWords)-1] = alias
			candidates[strings.Join(aliasWords, " ")] = path
		}
	}
	candidates["ops analyze"] = "ops analyse"
	candidates["ops analyze slow-process-instances"] = "ops analyse slow-process-instances"
	return candidates
}

func exampleIsDestructive(commandPath string, raw string) bool {
	if entry, ok := commandCoverageManifest[commandPath]; ok {
		return entry.Destructive
	}
	lower := strings.ToLower(raw)
	for _, token := range []string{" cancel ", " delete ", " deploy ", " run ", " update ", " resolve ", " purge ", " repair ", " retention-policy", " smoke-test"} {
		if strings.Contains(" "+lower+" ", token) {
			return true
		}
	}
	return false
}

func hasDestructiveWarning(source string) bool {
	lower := strings.ToLower(source)
	for _, term := range []string{"destructive", "--dry-run", "--auto-confirm", "confirmation", "without submitting mutation", "without submitting deletion", "without cancelling", "disposable"} {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

func generatedDestructiveWarning(commandPath string) string {
	if commandPath == "" {
		return ""
	}
	entry, ok := commandCoverageManifest[commandPath]
	if !ok || !entry.Destructive {
		return ""
	}
	return "destructive example; execute only against selected disposable integration profiles"
}

func exampleSourceText(t *testing.T, example cliExample) string {
	t.Helper()
	if example.SourceText != "" {
		return example.SourceText
	}
	data, err := os.ReadFile(example.SourcePath)
	if err != nil {
		t.Fatalf("read example source %s: %v", example.SourcePath, err)
	}
	return string(data)
}

func destructiveWarningSource(source string) string {
	if hasDestructiveWarning(source) {
		return "source"
	}
	return ""
}

func commandPathFromDocPath(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), ".md")
	if name == "c8volt" {
		return ""
	}
	name = strings.TrimPrefix(name, "c8volt_")
	return strings.ReplaceAll(name, "_", " ")
}

func summarizeExampleRecords(records []exampleValidationRecord) examplesSummary {
	summary := examplesSummary{Total: len(records)}
	for _, record := range records {
		switch record.SourceKind {
		case exampleSourceHelp:
			summary.HelpSources++
		case exampleSourceDocs:
			summary.DocSources++
		}
		if record.Outcome == "pass" {
			summary.Executed++
		}
		if len(record.Substitutions) > 0 {
			summary.Substituted++
		}
		if record.Destructive {
			summary.Destructive++
			if record.WarningPresent {
				summary.DestructiveWarnings++
			}
		}
		switch record.Outcome {
		case "skipped":
			summary.Skipped++
		case "blocked":
			summary.Blocked++
		case "fail":
			summary.Failed++
		}
	}
	return summary
}

func sortedManifestPaths() []string {
	paths := make([]string, 0, len(commandCoverageManifest))
	for path := range commandCoverageManifest {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func isExampleCommand(line string) bool {
	return strings.HasPrefix(line, "./c8volt") ||
		strings.HasPrefix(line, "c8volt ") ||
		strings.HasPrefix(line, "printf ")
}

func isC8VoltToken(token string) bool {
	return token == "c8volt" || token == "./c8volt" || strings.HasSuffix(token, "/c8volt")
}

func isRootFlag(arg string) bool {
	if !strings.HasPrefix(arg, "-") {
		return false
	}
	switch strings.TrimPrefix(strings.SplitN(arg, "=", 2)[0], "--") {
	case "auto-confirm", "automation", "debug", "help", "json", "keys-only", "no-indicator", "quiet", "verbose", "config", "log-level", "profile", "tenant", "timeout":
		return true
	default:
		return false
	}
}

func rootFlagConsumesValue(arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}
	switch strings.TrimPrefix(arg, "--") {
	case "config", "log-level", "profile", "tenant", "timeout":
		return true
	default:
		return false
	}
}

func hasExplicitConfigArg(args []string) bool {
	return rejectExplicitConfigArgs(args) != nil
}

func hasExplicitProfileArg(args []string) bool {
	for _, arg := range args {
		if arg == "--profile" || strings.HasPrefix(arg, "--profile=") {
			return true
		}
	}
	return false
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want || strings.HasPrefix(arg, want+"=") {
			return true
		}
	}
	return false
}

func hasHardCodedDemoSelector(command string) bool {
	return strings.Contains(command, "OrderProcess") || strings.Contains(command, "225179981")
}

func unresolvedPlaceholders(command string) []string {
	matches := regexp.MustCompile(`<[^>]+>`).FindAllString(command, -1)
	return copyNonEmptyStrings(matches)
}

func firstString(values []string) string {
	return stringAt(values, 0)
}

func isNumericString(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func stringAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

// stringAtOrFirst keeps multi-key examples runnable even when the suite seed produced one process instance.
func stringAtOrFirst(values []string, index int) string {
	if value := stringAt(values, index); value != "" {
		return value
	}
	return firstString(values)
}

func displayExampleSourcePath(sourcePath string) string {
	if strings.HasPrefix(sourcePath, "help:") {
		return sourcePath
	}
	if rel, err := filepath.Rel(suite.repoRoot, sourcePath); err == nil {
		return rel
	}
	return sourcePath
}

func hasDisposableTargetExecution(records []exampleValidationRecord) bool {
	for _, record := range records {
		if record.Outcome == "pass" && strings.HasPrefix(record.ExecutionMode, "disposable") {
			return true
		}
	}
	return false
}

// exampleNonExecutionAllowed documents the narrow cases that remain actionable evidence instead of hard failures.
func exampleNonExecutionAllowed(record exampleValidationRecord) bool {
	reason := strings.ToLower(record.FailureReason)
	switch {
	case strings.Contains(reason, "pipeline examples are recorded"):
		return true
	case strings.Contains(reason, "stdin-producing shell examples are recorded"):
		return true
	case strings.Contains(reason, "example passes --config"):
		return true
	case strings.Contains(reason, "documentation profile"):
		return true
	case strings.Contains(reason, "hard-coded documentation selectors"):
		return true
	case strings.Contains(reason, "unresolved placeholders"):
		return unresolvedPlaceholdersAllowed(record.UnresolvedPlaceholders)
	case strings.Contains(reason, "requires selected disposable profile"):
		return true
	case strings.Contains(reason, "mutating example requires a dedicated disposable scenario"):
		return true
	default:
		return false
	}
}

// unresolvedPlaceholdersAllowed documents setup states this suite records as blocked until fixture proposals exist.
func unresolvedPlaceholdersAllowed(placeholders []string) bool {
	allowed := map[string]struct{}{
		"<job-key>":              {},
		"<incident-key>":         {},
		"<another-incident-key>": {},
		"<resource-id>":          {},
		"<element-instance-key>": {},
		"<element-id>":           {},
		"<user-task-key>":        {},
		"<tenant-id>":            {},
	}
	for _, placeholder := range placeholders {
		if _, ok := allowed[placeholder]; !ok {
			return false
		}
	}
	return len(placeholders) > 0
}

func assertHasExampleFromSource(t *testing.T, examples []cliExample, sourceName string) {
	t.Helper()
	for _, example := range examples {
		if filepath.Base(example.SourcePath) == sourceName {
			return
		}
	}
	t.Fatalf("no examples from %s", sourceName)
}
