package cmd

import (
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
	case ModeJSON:
		cmd.Println(toolx.ToJSONString(items))
	case ModeKeysOnly:
		cmd.Println(strings.Join(mapItems(items, func(it process.ProcessInstance) string { return it.Key }), "\n"))
	default: // ModeOneLine
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
