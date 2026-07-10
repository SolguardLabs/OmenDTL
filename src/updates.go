package main

type DemandUpdater struct {
	Ledger *Ledger
}

func NewDemandUpdater(ledger *Ledger) *DemandUpdater {
	return &DemandUpdater{Ledger: ledger}
}

func (u *DemandUpdater) Apply(signal DemandSignal) (*DemandUpdate, error) {
	ledger := u.Ledger
	if ledger.ActiveCycle == nil || ledger.ActiveCycle.Status == CycleClosed {
		return nil, ErrNoOpenCycle
	}
	route, err := ledger.Route(signal.Route)
	if err != nil {
		return nil, err
	}
	if !signal.IsTrusted(route.Policy) {
		update := &DemandUpdate{
			ID:             ledger.NextDemandUpdateID(ledger.ActiveCycle.ID, route.ID),
			Cycle:          ledger.ActiveCycle.ID,
			Route:          route.ID,
			Source:         signal.Source,
			PreviousDemand: route.ObservedDemand,
			NextDemand:     signal.Amount,
			ForecastBasis:  route.ForecastedDemand,
			ConfidenceBps:  signal.Confidence,
			Status:         DemandUpdateRejected,
			Clock:          ledger.Clock,
			Reason:         "confidence below route floor",
		}
		ledger.DemandUpdates[update.ID] = update
		ledger.Events.Add(ledger.Clock, "demand.rejected", update.ID.String(), route.Asset, signal.Amount, map[string]interface{}{
			"route": route.ID.String(),
		})
		return update, ErrDemandUpdateRejected
	}
	forecast, err := ledger.ForecastForRoute(route.ID)
	if err != nil {
		return nil, err
	}
	source, err := ledger.Vault(route.SourceVault)
	if err != nil {
		return nil, err
	}
	previous := route.ObservedDemand
	route.MarkDemand(signal.Amount, ledger.Clock)
	source.LastDemandClock = ledger.Clock
	ledger.Demand.Add(signal)
	variance := Amount(0)
	released := Amount(0)
	if forecast.ForecastAmount > signal.Amount {
		variance = forecast.ForecastAmount - signal.Amount
		threshold := forecast.ForecastAmount.MulBpsFloor(route.Policy.ReleaseThresholdBps)
		if variance > threshold {
			released = forecast.Release(variance, ledger.Clock)
			released = source.ReleaseForecast(released)
			route.MarkReleased(released)
			ledger.ActiveCycle.AddRelease(route.Asset, released)
		}
	} else if signal.Amount > forecast.ForecastAmount {
		variance = signal.Amount - forecast.ForecastAmount
		buffer := variance.MulBpsCeil(route.Policy.PositiveDriftBps)
		if buffer > 0 && source.FreeLiquidity() >= buffer {
			if err := source.CommitForecast(buffer); err == nil {
				forecast.CommittedAmount += buffer
				forecast.ForecastAmount += buffer
			}
		}
	}
	status := DemandUpdateObserved
	if released > 0 {
		status = DemandUpdateAccepted
	}
	update := &DemandUpdate{
		ID:              ledger.NextDemandUpdateID(ledger.ActiveCycle.ID, route.ID),
		Cycle:           ledger.ActiveCycle.ID,
		Route:           route.ID,
		Source:          signal.Source,
		PreviousDemand:  previous,
		NextDemand:      signal.Amount,
		ForecastBasis:   forecast.ForecastAmount,
		Variance:        variance,
		ReleasedToVault: released,
		ConfidenceBps:   signal.Confidence,
		Status:          status,
		Clock:           ledger.Clock,
	}
	ledger.DemandUpdates[update.ID] = update
	ledger.Events.Add(ledger.Clock, "demand.updated", update.ID.String(), route.Asset, signal.Amount, map[string]interface{}{
		"route":    route.ID.String(),
		"released": released.Int64(),
	})
	return update, nil
}

func (u *DemandUpdater) ApplyMany(signals []DemandSignal) ([]DemandUpdateID, []DemandUpdateID) {
	accepted := []DemandUpdateID{}
	rejected := []DemandUpdateID{}
	for _, signal := range signals {
		update, err := u.Apply(signal)
		if update == nil {
			continue
		}
		if err != nil {
			rejected = append(rejected, update.ID)
			continue
		}
		accepted = append(accepted, update.ID)
	}
	return accepted, rejected
}
