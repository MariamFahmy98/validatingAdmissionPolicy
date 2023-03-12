package main

import (
	"github.com/spf13/cobra"
)

// CLI ...
func main() {
	cli := &cobra.Command {
		Use:   "policy-cli",
	}

	cli.AddCommand(ApplyCommand())
	cli.Execute()
}