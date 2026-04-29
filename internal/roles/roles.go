package roles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	apiclient "github.com/gravitational/teleport/api/client"
	"github.com/gravitational/trace"
)

type Result struct {
	User            string   `json:"user"`
	BaseRoles       []string `json:"base_roles"`
	AccessListRoles []string `json:"access_list_roles"`
	EffectiveRoles  []string `json:"effective_roles"`
}

func Get(ctx context.Context, tc *apiclient.Client, username string) (*Result, error) {
	uls, err := tc.UserLoginStateClient().GetUserLoginState(ctx, username)
	if err != nil {
		if trace.IsNotFound(err) {
			return nil, fmt.Errorf("no login state found for %q — has the user logged in since access lists were configured?", username)
		}
		if trace.IsAccessDenied(err) {
			return nil, fmt.Errorf("access denied — your user lacks permission to read user_login_state resources\n\n" +
				"Create a role granting access and assign it to your user:\n\n" +
				"  kind: role\n" +
				"  version: v7\n" +
				"  metadata:\n" +
				"    name: troles-access\n" +
				"  spec:\n" +
				"    allow:\n" +
				"      rules:\n" +
				"        - resources: [user_login_state]\n" +
				"          verbs: [list, read]")
		}
		return nil, fmt.Errorf("getting user login state for %q: %w", username, err)
	}

	return &Result{
		User:            username,
		BaseRoles:       sortedCopy(uls.GetOriginalRoles()),
		AccessListRoles: sortedCopy(uls.GetAccessListRoles()),
		EffectiveRoles:  sortedCopy(uls.GetRoles()),
	}, nil
}

func (r *Result) PrintTable(w io.Writer, color bool) {
	dim := esc(color, "\033[2m")
	bold := esc(color, "\033[1m")
	green := esc(color, "\033[32m")
	reset := esc(color, "\033[0m")

	fmt.Fprintf(w, "%sUser%s  %s%s%s\n\n", dim, reset, bold, r.User, reset)

	// Column width: widest role name or header, whichever is larger.
	roleWidth := len("ROLE")
	for _, role := range r.EffectiveRoles {
		if len(role) > roleWidth {
			roleWidth = len(role)
		}
	}

	fmt.Fprintf(w, "%s%-*s  SOURCE%s\n", dim, roleWidth, "ROLE", reset)

	alSet := make(map[string]bool, len(r.AccessListRoles))
	for _, role := range r.AccessListRoles {
		alSet[role] = true
	}

	for _, role := range r.EffectiveRoles {
		if alSet[role] {
			fmt.Fprintf(w, "%s%-*s  access list%s\n", green, roleWidth, role, reset)
		} else {
			fmt.Fprintf(w, "%-*s  base\n", roleWidth, role)
		}
	}

	total := len(r.EffectiveRoles)
	alCount := len(r.AccessListRoles)
	baseCount := total - alCount
	fmt.Fprintf(w, "\n")
	if alCount > 0 {
		fmt.Fprintf(w, "%s%d roles  (%d base, %d from access lists)%s\n", dim, total, baseCount, alCount, reset)
	} else {
		fmt.Fprintf(w, "%s%d roles%s\n", dim, total, reset)
	}
}

func (r *Result) PrintJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func esc(enabled bool, code string) string {
	if enabled {
		return code
	}
	return ""
}

func sortedCopy(s []string) []string {
	c := make([]string, len(s))
	copy(c, s)
	sort.Strings(c)
	return c
}
