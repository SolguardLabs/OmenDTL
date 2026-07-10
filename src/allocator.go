package main

type AllocationMode string

const (
	AllocationConservative AllocationMode = "conservative"
	AllocationBalanced     AllocationMode = "balanced"
	AllocationPriority     AllocationMode = "priority"
)

type Allocator struct {
	Ledger *Ledger
}

func NewAllocator(ledger *Ledger) *Allocator {
	return &Allocator{Ledger: ledger}
}

func (a *Allocator) Build(mode AllocationMode, owner AccountID) *ReservationBatch {
	cycleID := CycleID("")
	if a.Ledger.ActiveCycle != nil {
		cycleID = a.Ledger.ActiveCycle.ID
	}
	batch := NewReservationBatch(cycleID, a.Ledger.Clock)
	if a.Ledger.ActiveCycle == nil {
		return batch
	}
	for _, id := range SortedRouteIDs(a.Ledger.Routes) {
		route := a.Ledger.Routes[id]
		forecast, err := a.Ledger.ForecastForRoute(route.ID)
		if err != nil || forecast == nil {
			continue
		}
		amount := forecast.AvailableForReservation()
		switch mode {
		case AllocationConservative:
			amount = amount.MulBpsFloor(6_000)
		case AllocationBalanced:
			amount = amount.MulBpsFloor(8_000)
		case AllocationPriority:
			if route.Priority >= 8 {
				amount = amount.MulBpsFloor(9_500)
			} else {
				amount = amount.MulBpsFloor(7_000)
			}
		}
		amount = amount.Clamp(0, route.Policy.MaxReservation)
		if amount < route.EffectiveMinReservation() {
			continue
		}
		batch.AddRequest(ReservationRequest{
			Route:  route.ID,
			Owner:  owner,
			Amount: amount,
			Kind:   ReservationForecast,
		})
	}
	return batch
}

func (a *Allocator) Execute(batch *ReservationBatch) *ReservationBatch {
	if batch == nil {
		return nil
	}
	for _, request := range batch.Requests {
		reservation, err := a.Ledger.ReserveRoute(request.Route, request.Owner, request.Amount, request.Kind)
		if err != nil {
			batch.AddRejected(request.Route)
			a.Ledger.Events.Add(a.Ledger.Clock, "reservation.rejected", request.Route.String(), "", request.Amount, map[string]interface{}{
				"reason": err.Error(),
			})
			continue
		}
		batch.AddAccepted(reservation.ID, reservation.Asset, reservation.Amount)
	}
	return batch
}

func (a *Allocator) ExecuteMode(mode AllocationMode, owner AccountID) *ReservationBatch {
	return a.Execute(a.Build(mode, owner))
}
