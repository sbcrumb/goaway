package profile

type ProfileDetail struct {
	ID        uint                  `json:"id"`
	Name      string                `json:"name"`
	IsDefault bool                  `json:"isDefault"`
	Sources   []ProfileSourceStatus `json:"sources"`
}

type ProfileSourceStatus struct {
	SourceID uint   `json:"sourceId"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Active   bool   `json:"active"`
}

type SubnetRule struct {
	ID        uint   `json:"id"`
	CIDR      string `json:"cidr"`
	ProfileID uint   `json:"profileId"`
	ProfileName string `json:"profileName"`
}
