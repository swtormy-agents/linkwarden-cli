package cli

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/linkwarden-cli/internal/linkwarden"
)

func (a *app) collectionsCmd() *cobra.Command {
	collections := &cobra.Command{
		Use:   "collections",
		Short: "List collections",
	}
	collections.AddCommand(a.collectionsListCmd())
	return collections
}

func (a *app) collectionsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodGet,
				Path:   "/api/v1/collections",
			})
		},
	}
}
