package cli

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/linkwarden-cli/internal/linkwarden"
)

func (a *app) tagsCmd() *cobra.Command {
	tags := &cobra.Command{
		Use:   "tags",
		Short: "List tags",
	}
	tags.AddCommand(a.tagsListCmd())
	return tags
}

func (a *app) tagsListCmd() *cobra.Command {
	var sort, cursor int
	var search string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			if sort > 0 {
				q.Set("sort", strconv.Itoa(sort))
			}
			if cursor > 0 {
				q.Set("cursor", strconv.Itoa(cursor))
			}
			if search != "" {
				q.Set("search", search)
			}
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodGet,
				Path:   "/api/v1/tags",
				Query:  q,
			})
		},
	}
	cmd.Flags().IntVar(&sort, "sort", 0, "sort order (0-5)")
	cmd.Flags().IntVar(&cursor, "cursor", 0, "pagination cursor")
	cmd.Flags().StringVar(&search, "search", "", "filter tags by name")
	return cmd
}
