package main

type DemandSource string

const (
	DemandSourceObserved DemandSource = "observed"
	DemandSourceMerchant DemandSource = "merchant"
	DemandSourceOracle   DemandSource = "oracle"
	DemandSourceOperator DemandSource = "operator"
)

type DemandSignal struct {
	Route      RouteID      `json:"route"`
	Source     DemandSource `json:"source"`
	Amount     Amount       `json:"amount"`
	Confidence Amount       `json:"confidence_bps"`
	ShockBps   Amount       `json:"shock_bps"`
	Season     string       `json:"season"`
	Clock      int          `json:"clock"`
}

func NewDemandSignal(route RouteID, source DemandSource, amount Amount, confidence Amount, clock int) DemandSignal {
	return DemandSignal{
		Route:      route,
		Source:     source,
		Amount:     amount,
		Confidence: confidence.Clamp(0, BasisPoint),
		Clock:      clock,
	}
}

func (s DemandSignal) WeightedAmount(policy RoutePolicy) Amount {
	confidence := MaxAmount(s.Confidence, policy.ConfidenceFloorBps)
	base := s.Amount.MulBpsFloor(confidence)
	if s.ShockBps > 0 {
		base += s.Amount.MulBpsFloor(s.ShockBps)
	}
	return base
}

func (s DemandSignal) IsTrusted(policy RoutePolicy) bool {
	return s.Confidence >= policy.ConfidenceFloorBps
}

type DemandUpdateStatus string

const (
	DemandUpdateAccepted DemandUpdateStatus = "accepted"
	DemandUpdateObserved DemandUpdateStatus = "observed"
	DemandUpdateRejected DemandUpdateStatus = "rejected"
)

type DemandUpdate struct {
	ID              DemandUpdateID     `json:"id"`
	Cycle           CycleID            `json:"cycle"`
	Route           RouteID            `json:"route"`
	Source          DemandSource       `json:"source"`
	PreviousDemand  Amount             `json:"previous_demand"`
	NextDemand      Amount             `json:"next_demand"`
	ForecastBasis   Amount             `json:"forecast_basis"`
	Variance        Amount             `json:"variance"`
	ReleasedToVault Amount             `json:"released_to_vault"`
	ConfidenceBps   Amount             `json:"confidence_bps"`
	Status          DemandUpdateStatus `json:"status"`
	Clock           int                `json:"clock"`
	Reason          string             `json:"reason,omitempty"`
}

func (u *DemandUpdate) Report() DemandUpdate {
	if u == nil {
		return DemandUpdate{}
	}
	return *u
}

type DemandBook struct {
	signals map[RouteID][]DemandSignal
}

func NewDemandBook() *DemandBook {
	return &DemandBook{signals: map[RouteID][]DemandSignal{}}
}

func (b *DemandBook) Add(signal DemandSignal) {
	b.signals[signal.Route] = append(b.signals[signal.Route], signal)
}

func (b *DemandBook) Latest(route RouteID) (DemandSignal, bool) {
	values := b.signals[route]
	if len(values) == 0 {
		return DemandSignal{}, false
	}
	return values[len(values)-1], true
}

func (b *DemandBook) Average(route RouteID, window int) Amount {
	values := b.signals[route]
	if len(values) == 0 {
		return 0
	}
	if window <= 0 || window > len(values) {
		window = len(values)
	}
	var total Amount
	start := len(values) - window
	for _, signal := range values[start:] {
		total += signal.Amount
	}
	return Amount(int64(total) / int64(window))
}

func (b *DemandBook) Confidence(route RouteID, window int) Amount {
	values := b.signals[route]
	if len(values) == 0 {
		return 0
	}
	if window <= 0 || window > len(values) {
		window = len(values)
	}
	var total Amount
	start := len(values) - window
	for _, signal := range values[start:] {
		total += signal.Confidence
	}
	return Amount(int64(total) / int64(window))
}

func (b *DemandBook) Count(route RouteID) int {
	return len(b.signals[route])
}

func (b *DemandBook) Reports() map[RouteID][]DemandSignal {
	out := map[RouteID][]DemandSignal{}
	for route, values := range b.signals {
		copied := make([]DemandSignal, 0, len(values))
		copied = append(copied, values...)
		out[route] = copied
	}
	return out
}
