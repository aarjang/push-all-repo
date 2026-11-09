package ui

import (
	"fmt"

	"github.com/aarjang/push-all-repo/internal/model"
)

// ReviewTitle is the heading of the review list; it shows how many
// repositories are currently ticked so an emptied list is obvious before
// enter is pressed.
func ReviewTitle(selected, total int) string {
	return fmt.Sprintf("Repositories to push (%d of %d selected)", selected, total)
}

// FormatReviewRow renders one line of the review list:
// "name  [branch]  ↑2 | up to date | no upstream | no remote | dirty | detached | merge in progress".
func FormatReviewRow(c model.Candidate, s Style) string {
	branch := c.Branch
	if branch == "" {
		branch = "detached"
	}
	var tags []string
	switch {
	case c.Branch == "":
		tags = append(tags, "detached")
	case c.InProgressOp != "":
		tags = append(tags, c.InProgressOp+" in progress")
	}
	if c.Branch != "" {
		switch {
		case !c.HasRemote():
			tags = append(tags, "no remote")
		case c.Upstream == "":
			tags = append(tags, "no upstream")
		case c.AheadCount > 0:
			tags = append(tags, fmt.Sprintf("%s%d", s.Sym.Ahead, c.AheadCount))
		default:
			tags = append(tags, "up to date")
		}
	}
	if c.Dirty {
		tags = append(tags, "dirty")
	}
	out := fmt.Sprintf("%s  [%s]", c.Name, branch)
	for _, t := range tags {
		out += "  " + t
	}
	return out
}
