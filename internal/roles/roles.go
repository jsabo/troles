package roles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

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

func (r *Result) PrintTable(w io.Writer) {
	fmt.Fprintf(w, "User:  %s\n\n", r.User)

	fmt.Fprintf(w, "Base roles:\n")
	printList(w, r.BaseRoles)

	fmt.Fprintf(w, "\nAccess list grants:\n")
	printList(w, r.AccessListRoles)

	fmt.Fprintf(w, "\nEffective roles:\n")
	if len(r.EffectiveRoles) == 0 {
		fmt.Fprintf(w, "  (none)\n")
	} else {
		fmt.Fprintf(w, "  %s\n", strings.Join(r.EffectiveRoles, ", "))
	}
}

func (r *Result) PrintJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func printList(w io.Writer, roles []string) {
	if len(roles) == 0 {
		fmt.Fprintf(w, "  (none)\n")
		return
	}
	for _, role := range roles {
		fmt.Fprintf(w, "  %s\n", role)
	}
}

func sortedCopy(s []string) []string {
	c := make([]string, len(s))
	copy(c, s)
	sort.Strings(c)
	return c
}
