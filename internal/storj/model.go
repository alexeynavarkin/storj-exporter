package storj

type SNOResponse struct {
	NodeID string `json:"nodeID"`

	StartedAt        string `json:"startedAt"`
	LastPingedAt     string `json:"lastPinged"`
	LastQuicPingedAt string `json:"lastQuicPingedAt"`

	Version        string `json:"version"`
	AllowedVersion string `json:"allowedVersion"`
	UpToDate       bool   `json:"upToDate"`
	QuicStatus     string `json:"quicStatus"`

	DiskSpace struct {
		Used        float64 `json:"used"`
		Available   float64 `json:"available"`
		Trash       float64 `json:"trash"`
		Allocated   float64 `json:"allocated"`
		Overused    float64 `json:"overused"`
		Reclaimable float64 `json:"reclaimable"`
	} `json:"diskSpace"`

	Bandwidth struct {
		Used      float64 `json:"used"`
		Available float64 `json:"available"`
	} `json:"bandwidth"`

	Satellites []struct {
		ID           string  `json:"id"`
		URL          string  `json:"url"`
		Disqualified *string `json:"disqualified"`
		Suspended    *string `json:"suspended"`
		VettedAt     *string `json:"vettedAt"`
	} `json:"satellites"`
}

type SNOSatteliteResponse struct {
	EgressSummary    int64   `json:"egressSummary"`
	IngressSummary   int64   `json:"ingressSummary"`
	BandwidthSummary int64   `json:"bandwidthSummary"`
	StorageSummary   float64 `json:"storageSummary"`
	AverageUsageBytes float64 `json:"averageUsageBytes"`

	Audits struct {
		AuditScore      float64 `json:"auditScore"`
		SuspensionScore float64 `json:"suspensionScore"`
		OnlineScore     float64 `json:"onlineScore"`
	} `json:"audits"`

	PriceModel struct {
		EgressBandwidth int64 `json:"EgressBandwidth"`
		RepairBandwidth int64 `json:"RepairBandwidth"`
		AuditBandwidth  int64 `json:"AuditBandwidth"`
		DiskSpace       int64 `json:"DiskSpace"`
	} `json:"priceModel"`
}

type payoutMonth struct {
	EgressBandwidth         float64 `json:"egressBandwidth"`
	EgressBandwidthPayout   float64 `json:"egressBandwidthPayout"`
	EgressRepairAudit       float64 `json:"egressRepairAudit"`
	EgressRepairAuditPayout float64 `json:"egressRepairAuditPayout"`
	DiskSpace               float64 `json:"diskSpace"`
	DiskSpacePayout         float64 `json:"diskSpacePayout"`
	HeldRate                float64 `json:"heldRate"`
	Payout                  float64 `json:"payout"`
	Held                    float64 `json:"held"`
}

type SNOPayoutResponse struct {
	CurrentMonth             payoutMonth `json:"currentMonth"`
	PreviousMonth            payoutMonth `json:"previousMonth"`
	CurrentMonthExpectations float64     `json:"currentMonthExpectations"`
}
