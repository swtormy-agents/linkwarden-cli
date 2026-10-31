package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/linkwarden-cli/internal/linkwarden"
)

func (a *app) linksCmd() *cobra.Command {
	links := &cobra.Command{
		Use:   "links",
		Short: "List, search, and manage saved links",
	}
	links.AddCommand(
		a.linksListCmd(),
		a.linksSearchCmd(),
		a.linksGetCmd(),
		a.linksCreateCmd(),
		a.linksUpdateCmd(),
		a.linksDeleteCmd(),
	)
	return links
}

func (a *app) linksListCmd() *cobra.Command {
	var sort, cursor int
	var collectionID, tagID int
	var pinnedOnly bool
	var searchQuery string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List links (deprecated API; prefer search for new integrations)",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{}
			if sort > 0 {
				q.Set("sort", strconv.Itoa(sort))
			}
			if cursor > 0 {
				q.Set("cursor", strconv.Itoa(cursor))
			}
			if collectionID > 0 {
				q.Set("collectionId", strconv.Itoa(collectionID))
			}
			if tagID > 0 {
				q.Set("tagId", strconv.Itoa(tagID))
			}
			if pinnedOnly {
				q.Set("pinnedOnly", "true")
			}
			if searchQuery != "" {
				q.Set("searchQueryString", searchQuery)
			}
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodGet,
				Path:   "/api/v1/links",
				Query:  q,
			})
		},
	}
	cmd.Flags().IntVar(&sort, "sort", 0, "sort order (API numeric code)")
	cmd.Flags().IntVar(&cursor, "cursor", 0, "pagination cursor")
	cmd.Flags().IntVar(&collectionID, "collection-id", 0, "filter by collection id")
	cmd.Flags().IntVar(&tagID, "tag-id", 0, "filter by tag id")
	cmd.Flags().BoolVar(&pinnedOnly, "pinned-only", false, "only pinned links")
	cmd.Flags().StringVar(&searchQuery, "q", "", "search query string filter")
	return cmd
}

func (a *app) linksSearchCmd() *cobra.Command {
	var sort, cursor int
	var qStr string
	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search links via /api/v1/search",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			phrase := qStr
			if len(args) == 1 {
				phrase = args[0]
			}
			q := url.Values{}
			if phrase != "" {
				q.Set("searchQueryString", phrase)
			}
			if sort > 0 {
				q.Set("sort", strconv.Itoa(sort))
			}
			if cursor > 0 {
				q.Set("cursor", strconv.Itoa(cursor))
			}
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodGet,
				Path:   "/api/v1/search",
				Query:  q,
			})
		},
	}
	cmd.Flags().StringVar(&qStr, "q", "", "search phrase (alternative to positional QUERY)")
	cmd.Flags().IntVar(&sort, "sort", 0, "sort order (API numeric code)")
	cmd.Flags().IntVar(&cursor, "cursor", 0, "pagination cursor")
	return cmd
}

func (a *app) linksGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Retrieve one link by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodGet,
				Path:   "/api/v1/links/" + args[0],
			})
		},
	}
}

func (a *app) linksCreateCmd() *cobra.Command {
	var bodyFile, bodyJSON string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a link",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("links create"); err != nil {
				return err
			}
			body, err := a.readBody(bodyFile, bodyJSON)
			if err != nil {
				return err
			}
			if len(body) == 0 {
				return usageErr("provide --json or --body with link fields")
			}
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodPost,
				Path:   "/api/v1/links",
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&bodyJSON, "json", "", "link JSON object")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON file (- for stdin)")
	return cmd
}

func (a *app) linksUpdateCmd() *cobra.Command {
	var bodyFile, bodyJSON string
	cmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update a link by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("links update"); err != nil {
				return err
			}
			body, err := a.readBody(bodyFile, bodyJSON)
			if err != nil {
				return err
			}
			if len(body) == 0 {
				return usageErr("provide --json or --body with fields to update")
			}
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodPut,
				Path:   "/api/v1/links/" + args[0],
				Body:   body,
			})
		},
	}
	cmd.Flags().StringVar(&bodyJSON, "json", "", "link JSON object")
	cmd.Flags().StringVar(&bodyFile, "body", "", "path to JSON file (- for stdin)")
	return cmd
}

func (a *app) linksDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a link by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireWrite("links delete"); err != nil {
				return err
			}
			return a.do(context.Background(), linkwarden.Request{
				Method: http.MethodDelete,
				Path:   "/api/v1/links/" + args[0],
			})
		},
	}
}

func (a *app) readBody(filePath, jsonInline string) ([]byte, error) {
	if jsonInline != "" {
		if !json.Valid([]byte(jsonInline)) {
			return nil, usageErr("invalid --json")
		}
		return []byte(jsonInline), nil
	}
	if filePath == "" {
		return nil, nil
	}
	var b []byte
	var err error
	if filePath == "-" {
		b, err = readAllLimit(a.in, 4<<20)
	} else {
		b, err = os.ReadFile(filePath)
	}
	if err != nil {
		return nil, runtimeErr(err)
	}
	if !json.Valid(b) {
		return nil, usageErr("body is not valid JSON")
	}
	return b, nil
}

func readAllLimit(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("body too large")
	}
	return data, nil
}
