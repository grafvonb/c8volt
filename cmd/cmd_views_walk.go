package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

type Chain map[string]process.ProcessInstance
type KeysPath []string

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
		cmd.Println(toolx.ToJSONString(items))
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
