package storj

type SNOResponse struct {
	StartedAt        string `json:"startedAT"`
	LastPingedAt     string `json:"lastPinged"`
	LastQuicPingedAt string `json:"lastQuicPingedAt"`

	Version string `json:"version"`

	DiskSpace struct {
		Available float64 `json:"available"`
		Overused  float64 `json:"overused"`
		Trash     float64 `json:"trash"`
		Used      float64 `json:"used"`
	} `json:"diskSpace"`

	Satellites []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"satellites"`
}

type SNOSatteliteResponse struct {
	EgressSummary  int     `json:"egressSummary"`
	IngressSummary int     `json:"ingressSummary"`
	StorageSummary float64 `jsong:"storageSummary"`
	Audits         struct {
		AuditScore      float32 `json:"auditScore"`
		SuspensionScore float32 `json:"suspensionScore"`
		OnlineScore     float32 `json:"onlineScore"`
	} `json:"audits"`
}
