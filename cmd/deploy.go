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
	Short: "Deploy state-changing resources such as BPMN definitions",
	Long: `Deploy state-changing resources such as BPMN definitions.

Use this command family when you want c8volt to upload deployable assets into Camunda.
Choose ` + "`deploy process-definition`" + ` for local files or ` + "`embed deploy`" + ` for
bundled fixtures. Child commands explain when deployment waits for confirmation by
default, how ` + "`--no-wait`" + ` changes the completion contract, and what to inspect next.`,
	Example: `  ./c8volt deploy process-definition --help
  ./c8volt deploy process-definition --file ./order-process.bpmn
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
