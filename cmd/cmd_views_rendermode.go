package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

type RenderMode int

const (
	RenderModeJSON RenderMode = iota
	RenderModeOneLine
	RenderModeKeysOnly
	RenderModeTree
)

func (m RenderMode) String() string {
	switch m {
	case RenderModeJSON:
		return "json"
	case RenderModeOneLine:
		return "one-line"
	case RenderModeKeysOnly:
		return "keys-only"
	case RenderModeTree:
		return "tree"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

func pickMode() RenderMode {
	switch {
	case flagViewAsJson:
		return RenderModeJSON
	case flagViewKeysOnly:
		return RenderModeKeysOnly
	case flagViewAsTree:
		return RenderModeTree
	default:
		return RenderModeOneLine
	}
}

func itemView[Item any](cmd *cobra.Command, item Item, mode RenderMode, oneLine func(Item) string, keyOf func(Item) string) error {
	switch mode {
	case RenderModeJSON:
		cmd.Println(toolx.ToJSONString(item))
	case RenderModeKeysOnly:
		cmd.Println(keyOf(item))
	default:
		cmd.Println(strings.TrimSpace(oneLine(item)))
	}
	return nil
}

func listOrJSON[Resp any, Item any](cmd *cobra.Command, resp Resp, items []Item, mode RenderMode, oneLine func(Item) string, keyOf func(Item) string) error {
	switch mode {
	case RenderModeJSON:
		cmd.Print(toolx.ToJSONString(resp))
	case RenderModeKeysOnly:
		for _, it := range items {
			cmd.Println(keyOf(it))
		}
	default: // RenderModeOneLine
		for _, it := range items {
			cmd.Println(strings.TrimSpace(oneLine(it)))
		}
		cmd.Println("found:", len(items))
	}
	return nil
}
