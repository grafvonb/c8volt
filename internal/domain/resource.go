package domain

type Resource struct {
	ID         string `json:"id,omitempty"`
	Key        string `json:"key,omitempty"`
	Name       string `json:"name,omitempty"`
	TenantId   string `json:"tenantId,omitempty"`
	Version    int32  `json:"version,omitempty"`
	VersionTag string `json:"versionTag,omitempty"`
}
