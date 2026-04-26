// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy BPMN resources to Camunda",
	Long: `Deploy BPMN resources to Camunda.

Use ` + "`deploy pd`" + ` for local BPMN files or stdin. Use ` + "`embed deploy`" + ` when
you want the bundled fixtures that ship with c8volt.`,
	Example: `  ./c8volt embed export --file processdefinitions/C88_SimpleUserTaskProcess.bpmn --out ./fixtures
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn --run
  ./c8volt embed deploy --all --run`,
	Aliases: []string{"dep"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"depliy", "deplou"},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	addBackoffFlagsAndBindings(deployCmd)
	setCommandMutation(deployCmd, CommandMutationStateChanging)
}

func validateFiles(files []string) error {
	if len(files) == 0 {
		return invalidFlagValuef("at least one --file required")
	}
	count := 0
	for _, f := range files {
		if f == "-" {
			count++
			if count > 1 {
				return invalidFlagValuef("only one '-' (stdin) allowed")
			}
		}
	}
	return nil
}

func loadResources(paths []string, in io.Reader) ([]resource.DeploymentUnitData, error) {
	var out []resource.DeploymentUnitData
	for _, p := range paths {
		var b []byte
		var name string
		if p == "-" {
			var err error
			b, err = io.ReadAll(in)
			if err != nil {
				return nil, err
			}
			name = "stdin"
		} else {
			var err error
			b, err = os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			name = filepath.Base(p)
		}
		ct := detectContentType(name, b)
		out = append(out, resource.DeploymentUnitData{
			Name:        name,
			ContentType: ct,
			Data:        b,
		})
	}
	return out, nil
}

func detectContentType(name string, data []byte) string {
	if ext := filepath.Ext(name); ext != "" {
		if c := mime.TypeByExtension(ext); c != "" {
			return c
		}
	}
	// Fallback: sniff first 512 bytes
	return http.DetectContentType(data)
}
