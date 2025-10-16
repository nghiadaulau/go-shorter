package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "click-collector",
		Short: "Collects click events",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("collector started (placeholder)")
			select {}
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
