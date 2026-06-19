package permissions

import (
	_ "embed"
	"encoding/json"
	"slices"

	"github.com/rs/zerolog/log"
)

//go:embed permissions.json
var permissionsData []byte

type Permission struct {
	Permissions []string `json:"permissions"`
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Skip        bool     `json:"skip"`
}

type PermissionData struct {
	Endpoints []Permission `json:"endpoints"`
	Skip      bool         `json:"skip"`
}

func (r *PermissionData) FindPermissions(path, method string) Permission {
	idx := slices.IndexFunc(r.Endpoints, func(rp Permission) bool {
		return rp.Path == path && rp.Method == method
	})

	if idx == -1 {
		return Permission{}
	}

	return r.Endpoints[idx]
}

func Get() *PermissionData {
	var permissions PermissionData

	err := json.Unmarshal(permissionsData, &permissions)
	if err != nil {
		log.Err(err).Msg("Failed to decode embedded permissions")

		return nil
	}

	log.Info().Int("endpoints", len(permissions.Endpoints)).Msg("Successfully loaded embedded permissions")

	return &permissions
}
