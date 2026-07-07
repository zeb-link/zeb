// Shared command helpers for resolving API context.
// Commands call these before using the HTTP client so auth, URL, and active
// space behavior stays consistent.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/kerns/zlink-zeb/internal/api"
	"github.com/kerns/zlink-zeb/internal/config"
)

type apiContext struct {
	Client  *api.Client
	SpaceID string
	APIURL  string
}

// resolveClient builds an authenticated client without requiring an active
// space. Space-scoped commands should use resolveAPIContext instead.
func resolveClient(root *rootOptions) (*api.Client, string, error) {
	key, err := config.ResolveAPIKey(root.APIKey)
	if err != nil {
		return nil, "", err
	}
	if key == "" {
		return nil, "", fmt.Errorf("not logged in; run zeb auth login")
	}
	apiURL, err := config.ResolveAPIURL(root.APIURL)
	if err != nil {
		return nil, "", err
	}
	return api.New(api.Options{APIURL: apiURL, APIKey: key}), apiURL, nil
}

func resolveAPIContext(ctx context.Context, root *rootOptions) (apiContext, error) {
	client, apiURL, err := resolveClient(root)
	if err != nil {
		return apiContext{}, err
	}
	spaceID, err := config.ResolveSpace(root.SpaceID)
	if err != nil {
		return apiContext{}, err
	}
	if spaceID == "" {
		return apiContext{}, fmt.Errorf("no active space; run zeb space use <space-id-or-name>")
	}
	if !strings.HasPrefix(spaceID, "spc_") {
		me, err := client.GetMe(ctx)
		if err != nil {
			return apiContext{}, err
		}
		space, err := resolveSpace(me.AccessibleSpaces, spaceID)
		if err != nil {
			return apiContext{}, err
		}
		spaceID = space.ID
	}
	return apiContext{
		Client:  client,
		SpaceID: spaceID,
		APIURL:  apiURL,
	}, nil
}

// resolveByIDOrName resolves user input against a fetched list: exact id
// match first, then unique name match; ambiguous names must fall back to ids.
// Space and collection resolution share this so their semantics can't drift.
func resolveByIDOrName[T any](items []T, input string, kind string, id func(T) string, name func(T) string) (T, error) {
	var zero T
	for _, item := range items {
		if id(item) == input {
			return item, nil
		}
	}
	var matches []T
	for _, item := range items {
		if name(item) == input {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return zero, fmt.Errorf("multiple %ss named %q; use the %s id", kind, input, kind)
	}
	return zero, fmt.Errorf("%s %q not found", kind, input)
}
