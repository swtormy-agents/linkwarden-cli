package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/linkwarden-cli/skills/linkwarden"
)

func (a *app) skillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print the agent SKILL.md bundled with this CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(a.out, linkwarden.SkillMD)
			return err
		},
	}
}
