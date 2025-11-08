package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

type RenderMode int

const (
	ModeJSON RenderMode = iota
	ModeOneLine
	ModeKeysOnly
)

func (m RenderMode) String() string {
	switch m {
	case ModeJSON:
		return "json"
	case ModeOneLine:
		return "one-line"
	case ModeKeysOnly:
		return "keys-only"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

func pickMode() RenderMode {
	switch {
	case flagViewAsJson:
		return ModeJSON
	case flagViewKeysOnly:
		return ModeKeysOnly
	default:
		return ModeOneLine
	}
}

func itemView[Item any](cmd *cobra.Command, item Item, mode RenderMode, oneLine func(Item) string, keyOf func(Item) string) error {
	switch mode {
	case ModeJSON:
		cmd.Println(toolx.ToJSONString(item))
	case ModeKeysOnly:
		cmd.Println(keyOf(item))
	default:
		cmd.Println(strings.TrimSpace(oneLine(item)))
	}
	return nil
}

func listOrJSON[Resp any, Item any](cmd *cobra.Command, resp Resp, items []Item, mode RenderMode, oneLine func(Item) string, keyOf func(Item) string) error {
	switch mode {
	case ModeJSON:
		cmd.Print(toolx.ToJSONString(resp))
	case ModeKeysOnly:
		for _, it := range items {
			cmd.Println(keyOf(it))
		}
	default: // ModeOneLine
		for _, it := range items {
			cmd.Println(strings.TrimSpace(oneLine(it)))
		}
		cmd.Println("found:", len(items))
	}
	return nil
}
