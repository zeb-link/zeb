// The object-scope filter flags — "which links" — shared by `zeb links query`
// and `zeb analytics`, so the same dimension is the same flag with the same
// help on both commands (name once, align everywhere). Command-specific flags
// (link click thresholds, analytics click dimensions) live with their commands.
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type objectScopeFlags struct {
	Status       string
	Created      string
	Edited       string
	Schedule     string
	CreatedVia   string
	Attribution  string
	InCollection bool
	Uncollected  bool
	TargetHost   []string
	Not          []string
}

// objectScopeFlagNames are the flags addObjectScopeFlags registers — used to
// detect whether any object-scope flag was set (e.g. for --filter exclusivity).
var objectScopeFlagNames = []string{
	"status", "created", "edited", "schedule", "created-via", "attribution",
	"in-collection", "uncollected", "target-host", "not",
}

func addObjectScopeFlags(cmd *cobra.Command, f *objectScopeFlags) {
	cmd.Flags().StringVar(&f.Status, "status", "", "status: "+strings.Join(filterStatusValues, " | "))
	cmd.Flags().StringVar(&f.Created, "created", "", "created within a window: "+strings.Join(filterWindowValues, " | "))
	cmd.Flags().StringVar(&f.Edited, "edited", "", "last edited within a window: "+strings.Join(filterWindowValues, " | "))
	cmd.Flags().StringVar(&f.Schedule, "schedule", "", "schedule state: "+strings.Join(filterScheduleValues, " | "))
	cmd.Flags().StringVar(&f.CreatedVia, "created-via", "", "creation source: "+strings.Join(filterCreatedViaValues, " | "))
	cmd.Flags().StringVar(&f.Attribution, "attribution", "", "attribution carried: "+strings.Join(filterAttributionVals, " | "))
	cmd.Flags().BoolVar(&f.InCollection, "in-collection", false, "only links that are in at least one collection")
	cmd.Flags().BoolVar(&f.Uncollected, "uncollected", false, "only links that are in no collection")
	cmd.Flags().StringSliceVar(&f.TargetHost, "target-host", nil, "destination hostname(s); repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&f.Not, "not", nil, "invert an object-scope dimension; repeatable (e.g. --not status). Fields: "+strings.Join(filterNegatableFields, ", "))
}

// hasCollection resolves the in-collection/uncollected pair to a tri-state
// (nil = unset). Setting both is a contradiction.
func (f *objectScopeFlags) hasCollection() (*bool, error) {
	if f.InCollection && f.Uncollected {
		return nil, fmt.Errorf("--in-collection and --uncollected are mutually exclusive")
	}
	if f.InCollection {
		yes := true
		return &yes, nil
	}
	if f.Uncollected {
		no := false
		return &no, nil
	}
	return nil, nil
}

// anyObjectScopeFlagSet reports whether any shared object-scope flag was set.
func anyObjectScopeFlagSet(cmd *cobra.Command) bool {
	for _, name := range objectScopeFlagNames {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}
