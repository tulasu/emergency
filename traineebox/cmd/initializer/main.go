package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"traineebox/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root := &cobra.Command{
		Use:           "initializer",
		Short:         "TraineeBox host operations",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newMigrateCmd(cfg), newSeedAdminCmd(cfg), newImportBankCmd(cfg), newImportScenariosCmd(cfg), newSeedDemoCmd(cfg))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
