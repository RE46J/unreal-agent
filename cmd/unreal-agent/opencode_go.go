package main

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/unreallabsai/unreal-agent/cmd/internal/agentrunner"
	"github.com/unreallabsai/unreal-agent/harness/credential"
	"github.com/unreallabsai/unreal-agent/harness/llm/clients/opencodego"
	"github.com/unreallabsai/unreal-agent/harness/modelcatalog"
	"github.com/unreallabsai/unreal-agent/harness/provider"
)

// openCodeGoKey is where the docs and /login tell users to save the key.
var openCodeGoKey = credential.Reference{Provider: opencodego.ProviderID, Method: credential.APIKey, ID: "primary"}

// readOpenCodeGoModels reads the public model list; tests replace it.
var readOpenCodeGoModels = func(ctx context.Context) modelcatalog.Catalog {
	return modelcatalog.ReadOpenCodeGo(ctx, &http.Client{Timeout: 20 * time.Second}, opencodego.ModelsURL, opencodego.OfferedModels())
}

// openCodeGoStore is the store OpenCode Go's key is read from: the explicit
// one, else the launcher's default when it is a directory now. It is looked
// up on each discovery and request, so a key saved after the Host started is
// found by /model refresh (docs/opencode-go.md).
func openCodeGoStore(explicit, fallback string) string {
	if explicit != "" || fallback == "" {
		return explicit
	}
	if info, err := os.Lstat(fallback); err == nil && info.IsDir() {
		return fallback
	}
	return ""
}

// withDefaultOpenCodeGoStore resolves OpenCode Go's key from the default
// store, which is never created here; other references go to base.
func withDefaultOpenCodeGoStore(base credential.Resolver, fallback string) credential.Resolver {
	return credential.ResolverFunc(func(ctx context.Context, ref credential.Reference) (credential.Material, error) {
		if ref.Provider != opencodego.ProviderID {
			if base == nil {
				return credential.Material{}, &credential.Error{Code: "store_required"}
			}
			return base.Resolve(ctx, ref)
		}
		directory := openCodeGoStore("", fallback)
		if directory == "" {
			return credential.Material{}, &credential.Error{Code: "store_required"}
		}
		store, err := credential.OpenLocal(directory)
		if err != nil {
			return credential.Material{}, err
		}
		return credential.NewManager(store).Resolve(ctx, ref)
	})
}

// openCodeGoKeyStored reads the store's metadata only, and never creates it.
func openCodeGoKeyStored(ctx context.Context, directory string) (bool, error) {
	if directory == "" {
		return false, nil
	}
	if _, err := os.Lstat(directory); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	store, err := credential.OpenLocal(directory)
	if err != nil {
		return false, err
	}
	entries, err := store.List(ctx)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(entries, func(m credential.Metadata) bool { return m.Reference == openCodeGoKey }), nil
}

func discoverOpenCodeGo(ctx context.Context) (agentrunner.ProviderRuntime, error) {
	model, err := initialCatalogModel(readOpenCodeGoModels(ctx), "")
	if err != nil {
		return agentrunner.ProviderRuntime{}, &normalRuntimeError{Code: "catalog_unavailable", Provider: opencodego.ProviderID}
	}
	return agentrunner.ProviderRuntime{Provider: provider.Selection{Version: 1, Provider: opencodego.ProviderID, Model: provider.Model{ID: model.ID}, Endpoint: opencodego.BaseURL, Auth: openCodeGoKey, MaxAttempts: opencodego.DefaultMaxAttempts, MaxStreamAttempts: opencodego.DefaultMaxStreamAttempts, Source: "OpenCode Go model list"}, ReasoningEffort: model.DefaultEffort}, nil
}

// usesOpenCodeGo reports a registered OpenCode Go route, or a stored key whose
// model list could not be read yet.
func usesOpenCodeGo(config serveConfiguration) bool {
	return config.Runtime.Provider.Provider == opencodego.ProviderID || config.Providers[opencodego.ProviderID].Provider.Provider != "" ||
		slices.ContainsFunc(config.providerStatuses, func(p modelcatalog.Provider) bool { return p.ID == opencodego.ProviderID })
}

// withOpenCodeGoOrigin allows the OpenCode Go endpoint at the start of every
// normal Host, like Codex's, rather than rewriting the configuration file:
// restricted permission modes need it for model requests, and a key saved
// later must not need a restart. Nothing is sent without a key
// (docs/opencode-go.md).
func withOpenCodeGoOrigin(config serveConfiguration) serveConfiguration {
	u, err := url.Parse(opencodego.BaseURL)
	if err != nil {
		return config
	}
	origin := u.Scheme + "://" + u.Host
	if !slices.Contains(config.Permissions.NetworkOrigins, origin) {
		config.Permissions.NetworkOrigins = append(slices.Clone(config.Permissions.NetworkOrigins), origin)
	}
	return config
}

// openCodeGoKeyCommand is the command /login shows for saving the key. The
// launcher reads this store by default; the key comes from the clipboard so it
// never shows on screen or in the shell history.
func openCodeGoKeyCommand(getenv func(string) string) string {
	auth := "unreal-agent-auth"
	if executable, err := os.Executable(); err == nil {
		auth = filepath.Join(filepath.Dir(executable), "unreal-agent-auth")
	}
	store := "~/.local/state/unreal-agent/credentials"
	if root := getenv("XDG_STATE_HOME"); root != "" {
		store = filepath.Join(root, "unreal-agent", "credentials")
	}
	return "pbpaste | " + shellWord(auth) + " login --store " + shellWord(store) + " --provider " + openCodeGoKey.Provider + " --id " + openCodeGoKey.ID + " --key-stdin"
}

// shellWord quotes a path unless every character is safe in a shell word.
func shellWord(s string) string {
	unsafe := func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_./~:-", r))
	}
	if s != "" && strings.IndexFunc(s, unsafe) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// openCodeGoCatalog caches the model list for /model like claudeCatalog.
type openCodeGoCatalog struct {
	mu     sync.Mutex
	at     time.Time
	result modelcatalog.Catalog
}

func (c *openCodeGoCatalog) read(ctx context.Context, refresh bool) (modelcatalog.Catalog, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if refresh || c.at.IsZero() || time.Since(c.at) > time.Minute {
		c.result, c.at = readOpenCodeGoModels(ctx), time.Now()
	}
	return c.result.Clone(), nil
}
