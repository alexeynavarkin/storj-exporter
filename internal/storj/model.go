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

/*
	{
	     "currentMonth": {
	         "egressBandwidth": 174027539178,
	         "egressBandwidthPayout": 34.81,
	         "egressRepairAudit": 75179318272,
	         "egressRepairAuditPayout": 15.04,
	         "diskSpace": 1385674931617.358,
	         "diskSpacePayout": 206.48,
	         "heldRate": 0,
	         "payout": 256.33,
	         "held": 0
	     },
	     "previousMonth": {
	         "egressBandwidth": 114935609922,
	         "egressBandwidthPayout": 22.990000000000002,
	         "egressRepairAudit": 132170871690,
	         "egressRepairAuditPayout": 26.43,
	         "diskSpace": 2974113962562.14,
	         "diskSpacePayout": 443.14,
	         "heldRate": 0,
	         "payout": 492.56,
	         "held": 0
	     },
	     "currentMonthExpectations": 505
	 }
*/
type SNOPayoutResponse struct {
	CurrentMonth struct {
		Payout                  float32 `json:"payout"`
		DiskSpacePayout         float32 `json:"diskSpacePayout"`
		EgressBandwidthPayout   float32 `json:"egressBandwidthPayout"`
		EgressRepairAuditPayout float32 `json:"egressRepairAuditPayout"`
		Held                    float32 `json:"held"`
	}
	CurrentMonthExpectations float32 `json:"currentMonthExpectations"`
}
