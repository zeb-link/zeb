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

func resolveAPIContext(root *rootOptions) (apiContext, error) {
	key, err := config.ResolveAPIKey(root.APIKey)
	if err != nil {
		return apiContext{}, err
	}
	if key == "" {
		return apiContext{}, fmt.Errorf("not logged in; run zeb auth login")
	}
	apiURL, err := config.ResolveAPIURL(root.APIURL)
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
	client := api.New(api.Options{APIURL: apiURL, APIKey: key})
	if !strings.HasPrefix(spaceID, "spc_") {
		me, err := client.GetMe(context.Background())
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
