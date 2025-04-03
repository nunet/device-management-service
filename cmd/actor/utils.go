package actor

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func displayResponse(cmd *cobra.Command, resp any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(resp)
}
