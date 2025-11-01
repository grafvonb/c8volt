package cmd

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagDeployTenantId string
)

var deployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy resources",
	Aliases: []string{"dep"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"depliy", "deplou"},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	addBackoffFlagsAndBindings(deployCmd, viper.GetViper())

	deployCmd.PersistentFlags().StringVarP(&flagDeployTenantId, "tenant-id", "t", "", "tenant id for the deployment")
}

func validateFiles(files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("at least one --file required")
	}
	count := 0
	for _, f := range files {
		if f == "-" {
			count++
			if count > 1 {
				return fmt.Errorf("only one '-' (stdin) allowed")
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
