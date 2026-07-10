package main

type CycleOperator struct {
	Ledger *Ledger
}

func NewCycleOperator(ledger *Ledger) *CycleOperator {
	return &CycleOperator{Ledger: ledger}
}

func (o *CycleOperator) OpenAndAllocate(window int, mode AllocationMode, owner AccountID) (*ForecastCycle, *ReservationBatch, error) {
	cycle, err := o.Ledger.OpenForecastCycle(window)
	if err != nil {
		return nil, nil, err
	}
	batch := NewAllocator(o.Ledger).ExecuteMode(mode, owner)
	return cycle, batch, nil
}

func (o *CycleOperator) TickAndSettle(ticks int) SettlementResult {
	o.Ledger.Advance(ticks)
	return o.Ledger.FinalizeReadySettlements()
}

func (o *CycleOperator) DrainUntilIdle(maxTicks int) []SettlementResult {
	results := []SettlementResult{}
	for i := 0; i < maxTicks; i++ {
		result := o.TickAndSettle(1)
		results = append(results, result)
		if o.OpenSettlements() == 0 && o.OpenReservations() == 0 {
			break
		}
	}
	return results
}

func (o *CycleOperator) OpenReservations() int {
	count := 0
	for _, reservation := range o.Ledger.Reservations {
		if reservation.IsOpen() {
			count++
		}
	}
	return count
}

func (o *CycleOperator) OpenSettlements() int {
	count := 0
	for _, settlement := range o.Ledger.Settlements {
		if settlement.Status == SettlementPending {
			count++
		}
	}
	return count
}

func (o *CycleOperator) Close() error {
	return o.Ledger.CloseActiveCycle()
}
