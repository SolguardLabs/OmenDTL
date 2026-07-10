package main

type ForecastStatus string

const (
	ForecastOpen     ForecastStatus = "open"
	ForecastReserved ForecastStatus = "reserved"
	ForecastSettling ForecastStatus = "settling"
	ForecastClosed   ForecastStatus = "closed"
)

type ForecastEntry struct {
	ID              ForecastID     `json:"id"`
	Cycle           CycleID        `json:"cycle"`
	Route           RouteID        `json:"route"`
	SourceVault     VaultID        `json:"source_vault"`
	TargetVault     VaultID        `json:"target_vault"`
	Asset           AssetID        `json:"asset"`
	BasisDemand     Amount         `json:"basis_demand"`
	ForecastAmount  Amount         `json:"forecast_amount"`
	CommittedAmount Amount         `json:"committed_amount"`
	ReservedAmount  Amount         `json:"reserved_amount"`
	SettledAmount   Amount         `json:"settled_amount"`
	ReleasedAmount  Amount         `json:"released_amount"`
	FailedAmount    Amount         `json:"failed_amount"`
	ConfidenceBps   Amount         `json:"confidence_bps"`
	Status          ForecastStatus `json:"status"`
	CreatedClock    int            `json:"created_clock"`
	LastUpdateClock int            `json:"last_update_clock"`
}

func NewForecastEntry(cycle CycleID, route *Route, signal DemandSignal, asset *Asset, clock int) *ForecastEntry {
	weighted := signal.WeightedAmount(route.Policy)
	if weighted == 0 {
		weighted = route.BaseDemand.MulBpsFloor(route.Policy.ForecastWeightBps)
	}
	forecast := asset.Forecastable(weighted)
	if route.Policy.MaxReservation > 0 {
		forecast = MinAmount(forecast, route.Policy.MaxReservation)
	}
	forecast = MaxAmount(forecast, route.Policy.MinReservation)
	return &ForecastEntry{
		ID:              NewForecastID(cycle, route.ID),
		Cycle:           cycle,
		Route:           route.ID,
		SourceVault:     route.SourceVault,
		TargetVault:     route.TargetVault,
		Asset:           route.Asset,
		BasisDemand:     signal.Amount,
		ForecastAmount:  forecast,
		CommittedAmount: forecast,
		ConfidenceBps:   signal.Confidence,
		Status:          ForecastOpen,
		CreatedClock:    clock,
		LastUpdateClock: clock,
	}
}

func (f *ForecastEntry) Reserve(amount Amount, clock int) error {
	if f.Status == ForecastClosed {
		return ErrCycleClosed
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	available := f.AvailableForReservation()
	if available < amount {
		return ErrInsufficientLiquidity
	}
	f.ReservedAmount += amount
	f.LastUpdateClock = clock
	if f.ReservedAmount > 0 {
		f.Status = ForecastReserved
	}
	return nil
}

func (f *ForecastEntry) Release(amount Amount, clock int) Amount {
	if amount <= 0 {
		return 0
	}
	releasable := MinAmount(f.CommittedAmount, amount)
	f.CommittedAmount -= releasable
	f.ReleasedAmount += releasable
	f.LastUpdateClock = clock
	return releasable
}

func (f *ForecastEntry) MarkSettling(clock int) {
	if f.Status != ForecastClosed {
		f.Status = ForecastSettling
		f.LastUpdateClock = clock
	}
}

func (f *ForecastEntry) MarkSettled(amount Amount, clock int) {
	if amount <= 0 {
		return
	}
	if f.ReservedAmount >= amount {
		f.ReservedAmount -= amount
	}
	if f.CommittedAmount >= amount {
		f.CommittedAmount -= amount
	} else {
		f.CommittedAmount = 0
	}
	f.SettledAmount += amount
	f.LastUpdateClock = clock
	if f.ReservedAmount == 0 && f.CommittedAmount == 0 {
		f.Status = ForecastClosed
	}
}

func (f *ForecastEntry) MarkFailed(amount Amount, clock int) {
	if amount <= 0 {
		return
	}
	if f.ReservedAmount >= amount {
		f.ReservedAmount -= amount
	} else {
		f.ReservedAmount = 0
	}
	if f.CommittedAmount >= amount {
		f.CommittedAmount -= amount
	} else {
		f.CommittedAmount = 0
	}
	f.FailedAmount += amount
	f.LastUpdateClock = clock
	if f.ReservedAmount == 0 && f.CommittedAmount == 0 {
		f.Status = ForecastClosed
	}
}

func (f *ForecastEntry) AvailableForReservation() Amount {
	if f.ForecastAmount <= f.ReservedAmount+f.SettledAmount+f.FailedAmount {
		return 0
	}
	return f.ForecastAmount - f.ReservedAmount - f.SettledAmount - f.FailedAmount
}

func (f *ForecastEntry) ActiveBacking() Amount {
	return f.CommittedAmount + f.ReservedAmount
}

func (f *ForecastEntry) CloseRemainder(clock int) Amount {
	released := f.CommittedAmount
	f.ReleasedAmount += released
	f.CommittedAmount = 0
	f.ReservedAmount = 0
	f.Status = ForecastClosed
	f.LastUpdateClock = clock
	return released
}

type ForecastCycleStatus string

const (
	CycleOpen     ForecastCycleStatus = "open"
	CycleDraining ForecastCycleStatus = "draining"
	CycleClosed   ForecastCycleStatus = "closed"
)

type ForecastCycle struct {
	ID            CycleID             `json:"id"`
	WindowStart   int                 `json:"window_start"`
	WindowEnd     int                 `json:"window_end"`
	Status        ForecastCycleStatus `json:"status"`
	CreatedClock  int                 `json:"created_clock"`
	ClosedClock   int                 `json:"closed_clock,omitempty"`
	Forecasts     []ForecastID        `json:"forecasts"`
	ReleaseLedger AmountBuckets       `json:"release_ledger"`
}

func NewForecastCycle(id CycleID, start int, end int, clock int) *ForecastCycle {
	return &ForecastCycle{
		ID:            id,
		WindowStart:   start,
		WindowEnd:     end,
		Status:        CycleOpen,
		CreatedClock:  clock,
		Forecasts:     []ForecastID{},
		ReleaseLedger: AmountBuckets{},
	}
}

func (c *ForecastCycle) AddForecast(id ForecastID) {
	c.Forecasts = append(c.Forecasts, id)
}

func (c *ForecastCycle) MarkDraining() {
	if c.Status == CycleOpen {
		c.Status = CycleDraining
	}
}

func (c *ForecastCycle) AddRelease(asset AssetID, amount Amount) {
	c.ReleaseLedger.Add(asset, amount)
}

func (c *ForecastCycle) Close(clock int) {
	c.Status = CycleClosed
	c.ClosedClock = clock
}

type Forecaster struct {
	Ledger *Ledger
}

func NewForecaster(ledger *Ledger) *Forecaster {
	return &Forecaster{Ledger: ledger}
}

func (f *Forecaster) BuildSignals() []DemandSignal {
	signals := []DemandSignal{}
	for _, id := range SortedRouteIDs(f.Ledger.Routes) {
		route := f.Ledger.Routes[id]
		avg := f.Ledger.Demand.Average(route.ID, 4)
		confidence := f.Ledger.Demand.Confidence(route.ID, 4)
		if avg == 0 {
			avg = route.ObservedDemand
		}
		if confidence == 0 {
			confidence = route.Policy.ConfidenceFloorBps
		}
		signal := NewDemandSignal(route.ID, DemandSourceObserved, avg, confidence, f.Ledger.Clock)
		if route.Priority >= 8 {
			signal.ShockBps = route.Policy.SurgeWeightBps
		}
		signals = append(signals, signal)
	}
	return signals
}

func (f *Forecaster) CommitCycle(window int) (*ForecastCycle, error) {
	if f.Ledger.ActiveCycle != nil && f.Ledger.ActiveCycle.Status != CycleClosed {
		return nil, ErrCycleAlreadyOpen
	}
	cycle := NewForecastCycle(CycleID(NormalizeID("cycle", f.Ledger.NetworkID, f.Ledger.ClockString())), f.Ledger.Clock, f.Ledger.Clock+window, f.Ledger.Clock)
	signals := f.BuildSignals()
	for _, signal := range signals {
		route, err := f.Ledger.Route(signal.Route)
		if err != nil {
			return nil, err
		}
		if route.Status != RouteActive {
			continue
		}
		asset, err := f.Ledger.Assets.Get(route.Asset)
		if err != nil {
			return nil, err
		}
		source, err := f.Ledger.Vault(route.SourceVault)
		if err != nil {
			return nil, err
		}
		entry := NewForecastEntry(cycle.ID, route, signal, asset, f.Ledger.Clock)
		if entry.ForecastAmount == 0 {
			continue
		}
		if err := source.CommitForecast(entry.ForecastAmount); err != nil {
			entry.ForecastAmount = MinAmount(entry.ForecastAmount, source.FreeLiquidity())
			entry.CommittedAmount = entry.ForecastAmount
			if entry.ForecastAmount == 0 {
				continue
			}
			if commitErr := source.CommitForecast(entry.ForecastAmount); commitErr != nil {
				return nil, commitErr
			}
		}
		route.MarkForecast(entry.ForecastAmount, f.Ledger.Clock)
		f.Ledger.Forecasts[entry.ID] = entry
		cycle.AddForecast(entry.ID)
		f.Ledger.Events.Add(f.Ledger.Clock, "forecast.committed", entry.ID.String(), route.Asset, entry.ForecastAmount, map[string]interface{}{
			"route":        route.ID.String(),
			"source_vault": route.SourceVault.String(),
			"target_vault": route.TargetVault.String(),
		})
	}
	f.Ledger.ActiveCycle = cycle
	f.Ledger.Cycles[cycle.ID] = cycle
	return cycle, nil
}
