package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

type Chain map[string]process.ProcessInstance
type KeysPath []string

type walkTraversalPayload struct {
	Mode             process.TraversalMode     `json:"mode"`
	Outcome          process.TraversalOutcome  `json:"outcome"`
	RootKey          string                    `json:"rootKey,omitempty"`
	Keys             []string                  `json:"keys,omitempty"`
	Edges            map[string][]string       `json:"edges,omitempty"`
	Items            []process.ProcessInstance `json:"items,omitempty"`
	MissingAncestors []process.MissingAncestor `json:"missingAncestors,omitempty"`
	Warning          string                    `json:"warning,omitempty"`
}

func ancestorsView(cmd *cobra.Command, path KeysPath, chain Chain) error {
	return pathView(cmd, path, chain, pickMode(), " ← \n")
}

func descendantsView(cmd *cobra.Command, path KeysPath, chain Chain) error {
	return pathView(cmd, path, chain, pickMode(), " → \n")
}

func familyView(cmd *cobra.Command, path KeysPath, chain Chain) error {
	return pathView(cmd, path, chain, pickMode(), " ⇄ \n")
}

func pathView(cmd *cobra.Command, path KeysPath, chain Chain, mode RenderMode, sep string) error {
	items := pathItems(path, chain)
	switch mode {
	case RenderModeJSON:
		return renderJSONPayload(cmd, mode, items)
	case RenderModeKeysOnly:
		cmd.Println(strings.Join(mapItems(items, func(it process.ProcessInstance) string { return it.Key }), "\n"))
	default: // RenderModeOneLine
		cmd.Println(strings.Join(mapItems(items, oneLinePI), sep))
	}
	return nil
}

func pathItems(p KeysPath, c Chain) []process.ProcessInstance {
	out := make([]process.ProcessInstance, 0, len(p))
	for _, k := range p {
		if it, ok := c[k]; ok {
			out = append(out, it)
		}
	}
	return out
}

func mapItems[T any, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	for i := range in {
		out[i] = f(in[i])
	}
	return out
}

func traversalPayload(result process.TraversalResult) walkTraversalPayload {
	return walkTraversalPayload{
		Mode:             result.Mode,
		Outcome:          result.Outcome,
		RootKey:          result.RootKey,
		Keys:             append([]string(nil), result.Keys...),
		Edges:            result.Edges,
		Items:            pathItems(result.Keys, result.Chain),
		MissingAncestors: append([]process.MissingAncestor(nil), result.MissingAncestors...),
		Warning:          result.Warning,
	}
}

func printTraversalWarning(cmd *cobra.Command, result process.TraversalResult) {
	if result.Warning == "" && len(result.MissingAncestors) == 0 {
		return
	}

	warning := result.Warning
	if warning == "" {
		warning = "one or more parent process instances were not found"
	}
	cmd.Printf("warning: %s\n", warning)

	if len(result.MissingAncestors) == 0 {
		return
	}
	keys := make([]string, 0, len(result.MissingAncestors))
	for _, item := range result.MissingAncestors {
		keys = append(keys, item.Key)
	}
	cmd.Printf("missing ancestor keys: %s\n", strings.Join(keys, ", "))
}

// renderFamilyTree prints descendants as an ASCII tree starting from rootKey.
// It uses the edges map returned by Descendants/Family and the existing chain.
func renderFamilyTree(cmd *cobra.Command, rootKey string, edges map[string][]string, chain Chain, markerKey string) error {
	rootPI, ok := chain[rootKey]
	if !ok {
		return fmt.Errorf("root %s not found in chain", rootKey)
	}
	cmd.Println(oneLinePI(rootPI))
	var walk func(parentKey, prefix string)
	walk = func(parentKey, prefix string) {
		children := edges[parentKey]
		for i, childKey := range children {
			last := i == len(children)-1
			branch := "├─ "
			nextPrefix := prefix + "│  "
			if last {
				branch = "└─ "
				nextPrefix = prefix + "   "
			}
			pi, ok := chain[childKey]
			if !ok {
				continue
			}
			marker := ""
			if childKey == markerKey {
				marker = " (--key)"
			}
			cmd.Println(prefix + branch + oneLinePI(pi) + marker)
			walk(childKey, nextPrefix)
		}
	}
	walk(rootKey, "")
	return nil
}
