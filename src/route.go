package main

type RouteStatus string

const (
	RouteActive   RouteStatus = "active"
	RouteCooling  RouteStatus = "cooling"
	RouteDisabled RouteStatus = "disabled"
)

type RoutePolicy struct {
	MinReservation      Amount `json:"min_reservation"`
	MaxReservation      Amount `json:"max_reservation"`
	ForecastWeightBps   Amount `json:"forecast_weight_bps"`
	SurgeWeightBps      Amount `json:"surge_weight_bps"`
	ConfidenceFloorBps  Amount `json:"confidence_floor_bps"`
	SettlementDelay     int    `json:"settlement_delay"`
	WithdrawalCooldown  int    `json:"withdrawal_cooldown"`
	ReleaseThresholdBps Amount `json:"release_threshold_bps"`
	ReservationFeeBps   Amount `json:"reservation_fee_bps"`
	PositiveDriftBps    Amount `json:"positive_drift_bps"`
	NegativeDriftBps    Amount `json:"negative_drift_bps"`
}

func DefaultRoutePolicy() RoutePolicy {
	return RoutePolicy{
		MinReservation:      50_000,
		MaxReservation:      2_500_000,
		ForecastWeightBps:   9_000,
		SurgeWeightBps:      1_250,
		ConfidenceFloorBps:  6_500,
		SettlementDelay:     3,
		WithdrawalCooldown:  1,
		ReleaseThresholdBps: 300,
		ReservationFeeBps:   4,
		PositiveDriftBps:    1_500,
		NegativeDriftBps:    2_200,
	}
}

type Route struct {
	ID                  RouteID           `json:"id"`
	SourceVault         VaultID           `json:"source_vault"`
	TargetVault         VaultID           `json:"target_vault"`
	Asset               AssetID           `json:"asset"`
	Market              string            `json:"market"`
	BaseDemand          Amount            `json:"base_demand"`
	ObservedDemand      Amount            `json:"observed_demand"`
	ForecastedDemand    Amount            `json:"forecasted_demand"`
	Reserved            Amount            `json:"reserved"`
	Settled             Amount            `json:"settled"`
	Failed              Amount            `json:"failed"`
	ReleasedVariance    Amount            `json:"released_variance"`
	Priority            int               `json:"priority"`
	Status              RouteStatus       `json:"status"`
	Policy              RoutePolicy       `json:"policy"`
	LastForecastClock   int               `json:"last_forecast_clock"`
	LastDemandClock     int               `json:"last_demand_clock"`
	LastSettlementClock int               `json:"last_settlement_clock"`
	Tags                map[string]string `json:"tags"`
}

func NewRoute(id RouteID, source VaultID, target VaultID, asset AssetID, market string, demand Amount) *Route {
	return &Route{
		ID:             id,
		SourceVault:    source,
		TargetVault:    target,
		Asset:          asset,
		Market:         market,
		BaseDemand:     demand,
		ObservedDemand: demand,
		Priority:       1,
		Status:         RouteActive,
		Policy:         DefaultRoutePolicy(),
		Tags:           map[string]string{},
	}
}

func (r *Route) Clone() Route {
	copied := *r
	copied.Tags = map[string]string{}
	for key, value := range r.Tags {
		copied.Tags[key] = value
	}
	return copied
}

func (r *Route) OpenDemand() Amount {
	if r.ObservedDemand <= r.Settled+r.Failed {
		return 0
	}
	return r.ObservedDemand - r.Settled - r.Failed
}

func (r *Route) ForecastRemaining() Amount {
	if r.ForecastedDemand <= r.Reserved+r.Settled+r.Failed {
		return 0
	}
	return r.ForecastedDemand - r.Reserved - r.Settled - r.Failed
}

func (r *Route) ReservationCapacity() Amount {
	remaining := r.ForecastRemaining()
	if remaining == 0 {
		return 0
	}
	if r.Policy.MaxReservation > 0 {
		remaining = MinAmount(remaining, r.Policy.MaxReservation)
	}
	return remaining
}

func (r *Route) EffectiveMinReservation() Amount {
	if r.Policy.MinReservation <= 0 {
		return 1
	}
	return r.Policy.MinReservation
}

func (r *Route) CanReserve(amount Amount) bool {
	if r.Status != RouteActive {
		return false
	}
	if amount < r.EffectiveMinReservation() {
		return false
	}
	return r.ReservationCapacity() >= amount
}

func (r *Route) MarkForecast(amount Amount, clock int) {
	r.ForecastedDemand = amount
	r.LastForecastClock = clock
}

func (r *Route) MarkDemand(amount Amount, clock int) {
	r.ObservedDemand = amount
	r.LastDemandClock = clock
}

func (r *Route) MarkReserved(amount Amount) {
	if amount > 0 {
		r.Reserved += amount
	}
}

func (r *Route) MarkReleased(amount Amount) {
	if amount > 0 {
		r.ReleasedVariance += amount
	}
}

func (r *Route) MarkSettled(amount Amount, clock int) {
	if amount <= 0 {
		return
	}
	if r.Reserved >= amount {
		r.Reserved -= amount
	} else {
		r.Reserved = 0
	}
	r.Settled += amount
	r.LastSettlementClock = clock
}

func (r *Route) MarkFailed(amount Amount, clock int) {
	if amount <= 0 {
		return
	}
	if r.Reserved >= amount {
		r.Reserved -= amount
	} else {
		r.Reserved = 0
	}
	r.Failed += amount
	r.LastSettlementClock = clock
}

func (r *Route) DemandDrift() Amount {
	if r.ForecastedDemand >= r.ObservedDemand {
		return PercentageOf(r.ForecastedDemand-r.ObservedDemand, MaxAmount(r.ForecastedDemand, 1))
	}
	return PercentageOf(r.ObservedDemand-r.ForecastedDemand, MaxAmount(r.ForecastedDemand, 1))
}

type RouteReport struct {
	ID                  RouteID           `json:"id"`
	SourceVault         VaultID           `json:"source_vault"`
	TargetVault         VaultID           `json:"target_vault"`
	Asset               AssetID           `json:"asset"`
	Market              string            `json:"market"`
	BaseDemand          Amount            `json:"base_demand"`
	ObservedDemand      Amount            `json:"observed_demand"`
	ForecastedDemand    Amount            `json:"forecasted_demand"`
	OpenDemand          Amount            `json:"open_demand"`
	ForecastRemaining   Amount            `json:"forecast_remaining"`
	Reserved            Amount            `json:"reserved"`
	Settled             Amount            `json:"settled"`
	Failed              Amount            `json:"failed"`
	ReleasedVariance    Amount            `json:"released_variance"`
	Priority            int               `json:"priority"`
	Status              RouteStatus       `json:"status"`
	LastForecastClock   int               `json:"last_forecast_clock"`
	LastDemandClock     int               `json:"last_demand_clock"`
	LastSettlementClock int               `json:"last_settlement_clock"`
	Tags                map[string]string `json:"tags"`
}

func (r *Route) Report() RouteReport {
	tags := map[string]string{}
	for key, value := range r.Tags {
		tags[key] = value
	}
	return RouteReport{
		ID:                  r.ID,
		SourceVault:         r.SourceVault,
		TargetVault:         r.TargetVault,
		Asset:               r.Asset,
		Market:              r.Market,
		BaseDemand:          r.BaseDemand,
		ObservedDemand:      r.ObservedDemand,
		ForecastedDemand:    r.ForecastedDemand,
		OpenDemand:          r.OpenDemand(),
		ForecastRemaining:   r.ForecastRemaining(),
		Reserved:            r.Reserved,
		Settled:             r.Settled,
		Failed:              r.Failed,
		ReleasedVariance:    r.ReleasedVariance,
		Priority:            r.Priority,
		Status:              r.Status,
		LastForecastClock:   r.LastForecastClock,
		LastDemandClock:     r.LastDemandClock,
		LastSettlementClock: r.LastSettlementClock,
		Tags:                tags,
	}
}
