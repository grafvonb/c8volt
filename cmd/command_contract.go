package cmd

import (
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CommandMutation string

const (
	CommandMutationReadOnly      CommandMutation = "read_only"
	CommandMutationStateChanging CommandMutation = "state_changing"
)

type ContractSupport string

const (
	ContractSupportFull        ContractSupport = "full"
	ContractSupportLimited     ContractSupport = "limited"
	ContractSupportUnsupported ContractSupport = "unsupported"
)

type Outcome string

const (
	OutcomeSucceeded Outcome = "succeeded"
	OutcomeAccepted  Outcome = "accepted"
	OutcomeInvalid   Outcome = "invalid"
	OutcomeFailed    Outcome = "failed"
)

type CapabilityDocument struct {
	Command  string              `json:"command"`
	Version  string              `json:"version"`
	Commands []CommandCapability `json:"commands"`
}

type CommandCapability struct {
	Path            string               `json:"path"`
	Aliases         []string             `json:"aliases,omitempty"`
	Summary         string               `json:"summary"`
	Mutation        CommandMutation      `json:"mutation"`
	ContractSupport ContractSupport      `json:"contractSupport"`
	OutputModes     []OutputModeContract `json:"outputModes"`
	Flags           []FlagContract       `json:"flags,omitempty"`
	Children        []CommandCapability  `json:"children,omitempty"`
}

type FlagContract struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Repeated    bool   `json:"repeated"`
	Description string `json:"description,omitempty"`
}

type OutputModeContract struct {
	Name             string `json:"name"`
	Supported        bool   `json:"supported"`
	MachinePreferred bool   `json:"machinePreferred"`
	Notes            string `json:"notes,omitempty"`
}

type ResultDetail struct {
	Message    string `json:"message"`
	Class      string `json:"class,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type ResultEnvelope[T any] struct {
	Outcome Outcome       `json:"outcome"`
	Class   string        `json:"class,omitempty"`
	Command string        `json:"command"`
	Payload T             `json:"payload,omitempty"`
	Detail  *ResultDetail `json:"detail,omitempty"`
}

const (
	capabilityDocumentCommand = "capabilities"
	defaultContractVersion    = "v1"

	commandMutationAnnotation = "machine-contract/mutation"
	contractSupportAnnotation = "machine-contract/support"
	outputModesAnnotation     = "machine-contract/output-modes"
	contractVersionAnnotation = "machine-contract/version"
)

// setCapabilityDocumentVersion stores the discovery document version on the root command.
func setCapabilityDocumentVersion(cmd *cobra.Command, version string) {
	ensureCommandAnnotations(cmd)[contractVersionAnnotation] = version
}

// setCommandMutation records whether a command reads state or changes it.
func setCommandMutation(cmd *cobra.Command, mutation CommandMutation) {
	ensureCommandAnnotations(cmd)[commandMutationAnnotation] = string(mutation)
}

// commandMutationForCommand resolves the effective mutation classification for discovery output.
func commandMutationForCommand(cmd *cobra.Command) CommandMutation {
	if cmd == nil {
		return CommandMutationReadOnly
	}
	if value := strings.TrimSpace(cmd.Annotations[commandMutationAnnotation]); value != "" {
		return CommandMutation(value)
	}
	path := commandPath(cmd)
	if path == "" {
		return CommandMutationReadOnly
	}
	switch strings.Fields(path)[0] {
	case "get", "expect", "walk", "config", "version":
		return CommandMutationReadOnly
	default:
		return CommandMutationStateChanging
	}
}

// setContractSupport records how fully a command implements the shared machine contract.
func setContractSupport(cmd *cobra.Command, support ContractSupport) {
	ensureCommandAnnotations(cmd)[contractSupportAnnotation] = string(support)
}

// contractSupportForCommand resolves a command's support level, including inherited limited support from children.
func contractSupportForCommand(cmd *cobra.Command) ContractSupport {
	if cmd == nil {
		return ContractSupportUnsupported
	}
	if value := strings.TrimSpace(cmd.Annotations[contractSupportAnnotation]); value != "" {
		return ContractSupport(value)
	}
	for _, child := range cmd.Commands() {
		if !isDiscoverableCommand(child) {
			continue
		}
		if contractSupportForCommand(child) != ContractSupportUnsupported {
			return ContractSupportLimited
		}
	}
	return ContractSupportUnsupported
}

// setOutputModes stores explicit output-mode metadata for commands that need custom discovery reporting.
func setOutputModes(cmd *cobra.Command, modes ...OutputModeContract) {
	if cmd == nil {
		return
	}
	parts := make([]string, 0, len(modes))
	for _, mode := range modes {
		parts = append(parts, encodeOutputMode(mode))
	}
	ensureCommandAnnotations(cmd)[outputModesAnnotation] = strings.Join(parts, ",")
}

// outputModesForCommand returns explicit or inferred output modes for discovery consumers.
func outputModesForCommand(cmd *cobra.Command) []OutputModeContract {
	if cmd == nil {
		return nil
	}
	if value := strings.TrimSpace(cmd.Annotations[outputModesAnnotation]); value != "" {
		return decodeOutputModes(value)
	}

	support := contractSupportForCommand(cmd)
	modes := []OutputModeContract{{
		Name:      RenderModeOneLine.String(),
		Supported: true,
	}}
	if hasFlag(cmd, "json") {
		modes = append(modes, OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: support == ContractSupportFull,
		})
	}
	if hasFlag(cmd, "keys-only") {
		modes = append(modes, OutputModeContract{
			Name:      RenderModeKeysOnly.String(),
			Supported: true,
		})
	}
	if hasFlag(cmd, "tree") {
		modes = append(modes, OutputModeContract{
			Name:      RenderModeTree.String(),
			Supported: true,
		})
	}
	return modes
}

// capabilityDocumentForRoot builds the top-level discovery document from the live Cobra tree.
func capabilityDocumentForRoot(root *cobra.Command) CapabilityDocument {
	version := defaultContractVersion
	if root != nil {
		if value := strings.TrimSpace(root.Annotations[contractVersionAnnotation]); value != "" {
			version = value
		}
	}

	commands := make([]CommandCapability, 0)
	if root != nil {
		for _, child := range root.Commands() {
			if !isDiscoverableCommand(child) {
				continue
			}
			commands = append(commands, commandCapabilityForCommand(child))
		}
	}

	return CapabilityDocument{
		Command:  capabilityDocumentCommand,
		Version:  version,
		Commands: commands,
	}
}

// commandCapabilityForCommand converts one Cobra command and its discoverable children into capability metadata.
func commandCapabilityForCommand(cmd *cobra.Command) CommandCapability {
	children := make([]CommandCapability, 0)
	for _, child := range cmd.Commands() {
		if !isDiscoverableCommand(child) {
			continue
		}
		children = append(children, commandCapabilityForCommand(child))
	}

	return CommandCapability{
		Path:            commandPath(cmd),
		Aliases:         slices.Clone(cmd.Aliases),
		Summary:         strings.TrimSpace(cmd.Short),
		Mutation:        commandMutationForCommand(cmd),
		ContractSupport: contractSupportForCommand(cmd),
		OutputModes:     outputModesForCommand(cmd),
		Flags:           flagContractsForCommand(cmd),
		Children:        children,
	}
}

// flagContractsForCommand serializes visible flags into machine-readable flag metadata.
func flagContractsForCommand(cmd *cobra.Command) []FlagContract {
	if cmd == nil {
		return nil
	}

	flags := make([]FlagContract, 0)
	seen := make(map[string]struct{})
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag == nil || flag.Hidden {
			return
		}
		if _, ok := seen[flag.Name]; ok {
			return
		}
		seen[flag.Name] = struct{}{}
		flags = append(flags, FlagContract{
			Name:        flag.Name,
			Shorthand:   flag.Shorthand,
			Type:        flag.Value.Type(),
			Required:    isRequiredFlag(flag),
			Repeated:    isRepeatedFlagType(flag.Value.Type()),
			Description: strings.TrimSpace(flag.Usage),
		})
	})
	return flags
}

// commandPath returns the command path without the root binary name.
func commandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := cmd.CommandPath()
	root := Root().Name()
	if path == root {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(path, root))
}

// isDiscoverableCommand filters out hidden and shell-internal commands from capability output.
func isDiscoverableCommand(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Hidden {
		return false
	}
	if strings.HasPrefix(cmd.Name(), "__complete") {
		return false
	}
	return cmd.Name() != "help"
}

// hasFlag reports whether a command exposes a given flag after Cobra inheritance is applied.
func hasFlag(cmd *cobra.Command, name string) bool {
	return cmd != nil && cmd.Flags().Lookup(name) != nil
}

// isRequiredFlag reports whether Cobra marks a flag as required.
func isRequiredFlag(flag *pflag.Flag) bool {
	if flag == nil || flag.Annotations == nil {
		return false
	}
	_, ok := flag.Annotations[cobra.BashCompOneRequiredFlag]
	return ok
}

// isRepeatedFlagType identifies flag types that can appear multiple times in one invocation.
func isRepeatedFlagType(flagType string) bool {
	switch flagType {
	case "stringSlice", "stringArray", "intSlice", "durationSlice":
		return true
	default:
		return false
	}
}

// ensureCommandAnnotations lazily creates the annotation map used for machine-contract metadata.
func ensureCommandAnnotations(cmd *cobra.Command) map[string]string {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	return cmd.Annotations
}

// encodeOutputMode flattens one output-mode contract into a compact annotation string.
func encodeOutputMode(mode OutputModeContract) string {
	parts := []string{mode.Name}
	if mode.Supported {
		parts = append(parts, "supported")
	} else {
		parts = append(parts, "unsupported")
	}
	if mode.MachinePreferred {
		parts = append(parts, "preferred")
	}
	if mode.Notes != "" {
		parts = append(parts, "note="+strings.ReplaceAll(mode.Notes, ",", ";"))
	}
	return strings.Join(parts, "|")
}

// decodeOutputModes expands stored annotation text back into output-mode contract values.
func decodeOutputModes(raw string) []OutputModeContract {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	items := strings.Split(raw, ",")
	modes := make([]OutputModeContract, 0, len(items))
	for _, item := range items {
		parts := strings.Split(item, "|")
		if len(parts) == 0 {
			continue
		}
		mode := OutputModeContract{
			Name:      parts[0],
			Supported: len(parts) < 2 || parts[1] == "supported",
		}
		for _, part := range parts[2:] {
			switch {
			case part == "preferred":
				mode.MachinePreferred = true
			case strings.HasPrefix(part, "note="):
				mode.Notes = strings.TrimPrefix(part, "note=")
			}
		}
		modes = append(modes, mode)
	}
	return modes
}
