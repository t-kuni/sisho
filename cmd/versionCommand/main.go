package versionCommand

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version string
var Revision string

type VersionCommand struct {
	CobraCommand *cobra.Command
}

func NewVersionCommand() *VersionCommand {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of sisho",
		Long:  `All software has versions. This is sisho's`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("sisho version %s (rev: %s)\n", Version, Revision)
		},
	}

	return &VersionCommand{
		CobraCommand: cmd,
	}
}
