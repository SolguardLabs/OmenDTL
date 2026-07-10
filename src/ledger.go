package main

import "fmt"

type Ledger struct {
	NetworkID       string
	Clock           int
	Assets          *AssetRegistry
	Accounts        map[AccountID]*Account
	Vaults          map[VaultID]*Vault
	Routes          map[RouteID]*Route
	Forecasts       map[ForecastID]*ForecastEntry
	Reservations    map[ReservationID]*Reservation
	Settlements     map[SettlementID]*Settlement
	DemandUpdates   map[DemandUpdateID]*DemandUpdate
	Withdrawals     map[WithdrawalID]*Withdrawal
	Cycles          map[CycleID]*ForecastCycle
	Demand          *DemandBook
	Events          *EventLog
	ActiveCycle     *ForecastCycle
	Notes           []string
	reservationSeq  int
	demandUpdateSeq int
	withdrawalSeq   int
}

func NewLedger(network string) *Ledger {
	return &Ledger{
		NetworkID:     network,
		Assets:        NewAssetRegistry(),
		Accounts:      map[AccountID]*Account{},
		Vaults:        map[VaultID]*Vault{},
		Routes:        map[RouteID]*Route{},
		Forecasts:     map[ForecastID]*ForecastEntry{},
		Reservations:  map[ReservationID]*Reservation{},
		Settlements:   map[SettlementID]*Settlement{},
		DemandUpdates: map[DemandUpdateID]*DemandUpdate{},
		Withdrawals:   map[WithdrawalID]*Withdrawal{},
		Cycles:        map[CycleID]*ForecastCycle{},
		Demand:        NewDemandBook(),
		Events:        NewEventLog(),
		Notes:         []string{},
	}
}

func (l *Ledger) ClockString() string {
	return fmt.Sprintf("%06d", l.Clock)
}

func (l *Ledger) Advance(ticks int) {
	if ticks <= 0 {
		return
	}
	l.Clock += ticks
	l.Events.Add(l.Clock, "clock.advance", "", "", Amount(ticks), nil)
}

func (l *Ledger) AddNote(note string) {
	if note == "" {
		return
	}
	l.Notes = append(l.Notes, note)
}

func (l *Ledger) AddAsset(asset *Asset) {
	l.Assets.Add(asset)
	l.Events.Add(l.Clock, "asset.registered", asset.ID.String(), asset.ID, 0, map[string]interface{}{
		"symbol": asset.Symbol,
		"kind":   string(asset.Kind),
	})
}

func (l *Ledger) AddAccount(account *Account) {
	if account == nil {
		return
	}
	copied := account.Clone()
	l.Accounts[account.ID] = &copied
	l.Events.Add(l.Clock, "account.registered", account.ID.String(), "", 0, map[string]interface{}{
		"role": string(account.Role),
	})
}

func (l *Ledger) AddVault(vault *Vault) error {
	if vault == nil {
		return nil
	}
	if !l.Assets.Exists(vault.Asset) {
		return ErrUnknownAsset
	}
	copied := vault.Clone()
	l.Vaults[vault.ID] = &copied
	l.Events.Add(l.Clock, "vault.registered", vault.ID.String(), vault.Asset, vault.Reserve, map[string]interface{}{
		"region": vault.Region,
	})
	return nil
}

func (l *Ledger) AddRoute(route *Route) error {
	if route == nil {
		return nil
	}
	source, err := l.Vault(route.SourceVault)
	if err != nil {
		return err
	}
	target, err := l.Vault(route.TargetVault)
	if err != nil {
		return err
	}
	if source.Asset != route.Asset || target.Asset != route.Asset {
		return ErrInvalidRouteAsset
	}
	copied := route.Clone()
	l.Routes[route.ID] = &copied
	l.Demand.Add(NewDemandSignal(route.ID, DemandSourceObserved, route.ObservedDemand, route.Policy.ConfidenceFloorBps, l.Clock))
	l.Events.Add(l.Clock, "route.registered", route.ID.String(), route.Asset, route.BaseDemand, map[string]interface{}{
		"market":       route.Market,
		"source_vault": route.SourceVault.String(),
		"target_vault": route.TargetVault.String(),
	})
	return nil
}

func (l *Ledger) Account(id AccountID) (*Account, error) {
	account, ok := l.Accounts[id]
	if !ok {
		return nil, ErrUnknownAccount
	}
	return account, nil
}

func (l *Ledger) Vault(id VaultID) (*Vault, error) {
	vault, ok := l.Vaults[id]
	if !ok {
		return nil, ErrUnknownVault
	}
	return vault, nil
}

func (l *Ledger) Route(id RouteID) (*Route, error) {
	route, ok := l.Routes[id]
	if !ok {
		return nil, ErrUnknownRoute
	}
	return route, nil
}

func (l *Ledger) Forecast(id ForecastID) (*ForecastEntry, error) {
	forecast, ok := l.Forecasts[id]
	if !ok {
		return nil, ErrUnknownForecast
	}
	return forecast, nil
}

func (l *Ledger) Reservation(id ReservationID) (*Reservation, error) {
	reservation, ok := l.Reservations[id]
	if !ok {
		return nil, ErrUnknownReservation
	}
	return reservation, nil
}

func (l *Ledger) Settlement(id SettlementID) (*Settlement, error) {
	settlement, ok := l.Settlements[id]
	if !ok {
		return nil, ErrUnknownSettlement
	}
	return settlement, nil
}

func (l *Ledger) OpenForecastCycle(window int) (*ForecastCycle, error) {
	if window <= 0 {
		window = 6
	}
	cycle, err := NewForecaster(l).CommitCycle(window)
	if err != nil {
		return nil, WrapError("open forecast cycle", err)
	}
	l.Events.Add(l.Clock, "cycle.opened", cycle.ID.String(), "", Amount(len(cycle.Forecasts)), map[string]interface{}{
		"window_start": cycle.WindowStart,
		"window_end":   cycle.WindowEnd,
	})
	return cycle, nil
}

func (l *Ledger) ForecastForRoute(routeID RouteID) (*ForecastEntry, error) {
	if l.ActiveCycle == nil {
		return nil, ErrNoOpenCycle
	}
	for _, id := range l.ActiveCycle.Forecasts {
		forecast := l.Forecasts[id]
		if forecast.Route == routeID && forecast.Status != ForecastClosed {
			return forecast, nil
		}
	}
	return nil, ErrUnknownForecast
}

func (l *Ledger) NextReservationID(cycle CycleID, route RouteID) ReservationID {
	l.reservationSeq++
	return NewReservationID(cycle, route, l.reservationSeq)
}

func (l *Ledger) NextDemandUpdateID(cycle CycleID, route RouteID) DemandUpdateID {
	l.demandUpdateSeq++
	return NewDemandUpdateID(cycle, route, l.demandUpdateSeq)
}

func (l *Ledger) NextWithdrawalID(vault VaultID, account AccountID) WithdrawalID {
	l.withdrawalSeq++
	return NewWithdrawalID(vault, account, l.withdrawalSeq)
}

func (l *Ledger) ReserveRoute(routeID RouteID, owner AccountID, amount Amount, kind ReservationKind) (*Reservation, error) {
	if l.ActiveCycle == nil || l.ActiveCycle.Status == CycleClosed {
		return nil, ErrNoOpenCycle
	}
	route, err := l.Route(routeID)
	if err != nil {
		return nil, err
	}
	if _, err := l.Account(owner); err != nil {
		return nil, err
	}
	if !route.CanReserve(amount) {
		return nil, ErrInsufficientLiquidity
	}
	forecast, err := l.ForecastForRoute(routeID)
	if err != nil {
		return nil, err
	}
	if err := forecast.Reserve(amount, l.Clock); err != nil {
		return nil, err
	}
	asset := l.Assets.MustGet(route.Asset)
	fee := asset.SettlementFee(amount)
	id := l.NextReservationID(l.ActiveCycle.ID, route.ID)
	reservation := NewReservation(id, l.ActiveCycle.ID, forecast, route, owner, kind, amount, fee, l.Clock)
	l.Reservations[id] = reservation
	route.MarkReserved(amount)
	l.Events.Add(l.Clock, "reservation.queued", reservation.ID.String(), route.Asset, amount, map[string]interface{}{
		"route":      route.ID.String(),
		"forecast":   forecast.ID.String(),
		"settlement": reservation.SettlementDue,
	})
	return reservation, nil
}

func (l *Ledger) BindReservation(id ReservationID) error {
	reservation, err := l.Reservation(id)
	if err != nil {
		return err
	}
	if reservation.Status != ReservationQueued {
		return nil
	}
	source, err := l.Vault(reservation.SourceVault)
	if err != nil {
		return err
	}
	if err := source.AttachReservation(reservation.Amount); err != nil {
		return err
	}
	reservation.Bind(l.Clock)
	l.Events.Add(l.Clock, "reservation.bound", reservation.ID.String(), reservation.Asset, reservation.Amount, nil)
	return nil
}

func (l *Ledger) CreateSettlement(id ReservationID) (*Settlement, error) {
	reservation, err := l.Reservation(id)
	if err != nil {
		return nil, err
	}
	if !reservation.IsOpen() {
		return nil, ErrReservationClosed
	}
	if err := l.BindReservation(id); err != nil {
		return nil, err
	}
	reservation.MarkSettling(l.Clock)
	source, err := l.Vault(reservation.SourceVault)
	if err != nil {
		return nil, err
	}
	if err := source.OpenSettlementDebt(reservation.Amount); err != nil {
		reservation.Fail(l.Clock, err.Error())
		return nil, err
	}
	forecast, _ := l.Forecast(reservation.Forecast)
	if forecast != nil {
		forecast.MarkSettling(l.Clock)
	}
	settlement := NewSettlement(reservation, l.Clock)
	l.Settlements[settlement.ID] = settlement
	l.Events.Add(l.Clock, "settlement.pending", settlement.ID.String(), settlement.Asset, settlement.Amount, map[string]interface{}{
		"reservation": reservation.ID.String(),
		"due":         settlement.DueClock,
	})
	return settlement, nil
}

func (l *Ledger) EnsureSettlements() []SettlementID {
	created := []SettlementID{}
	for _, id := range SortedReservationIDs(l.Reservations) {
		reservation := l.Reservations[id]
		if reservation.Status == ReservationQueued || reservation.Status == ReservationBound {
			if _, exists := l.Settlements[NewSettlementID(id)]; exists {
				continue
			}
			if reservation.Ready(l.Clock) {
				settlement, err := l.CreateSettlement(id)
				if err == nil {
					created = append(created, settlement.ID)
				} else {
					l.Events.Add(l.Clock, "settlement.prepare_failed", id.String(), reservation.Asset, reservation.Amount, map[string]interface{}{
						"reason": err.Error(),
					})
				}
			}
		}
	}
	return created
}

func (l *Ledger) FinalizeReadySettlements() SettlementResult {
	l.EnsureSettlements()
	result := NewSettlementResult(l.Clock)
	for _, id := range SortedSettlementIDs(l.Settlements) {
		settlement := l.Settlements[id]
		if !settlement.Ready(l.Clock) {
			continue
		}
		source, sourceErr := l.Vault(settlement.SourceVault)
		target, targetErr := l.Vault(settlement.TargetVault)
		reservation, reservationErr := l.Reservation(settlement.Reservation)
		route, routeErr := l.Route(settlement.Route)
		if sourceErr != nil || targetErr != nil || reservationErr != nil || routeErr != nil {
			settlement.Reject(l.Clock, "missing settlement link")
			result.AddRejected(settlement.ID)
			continue
		}
		forecast, _ := l.Forecast(reservation.Forecast)
		if err := source.CompleteSettlementOut(settlement.Amount, l.Clock); err != nil {
			settlement.Freeze(l.Clock, err.Error())
			reservation.Fail(l.Clock, err.Error())
			route.MarkFailed(settlement.Amount, l.Clock)
			result.AddFrozen(settlement.ID)
			l.Events.Add(l.Clock, "settlement.frozen", settlement.ID.String(), settlement.Asset, settlement.Amount, map[string]interface{}{
				"reason": err.Error(),
			})
			continue
		}
		if err := target.ReceiveSettlement(settlement.NetAmount, l.Clock); err != nil {
			settlement.Freeze(l.Clock, err.Error())
			result.AddFrozen(settlement.ID)
			continue
		}
		if forecast != nil {
			forecast.MarkSettled(settlement.Amount, l.Clock)
		}
		route.MarkSettled(settlement.Amount, l.Clock)
		reservation.Complete(l.Clock)
		settlement.Finalize(l.Clock)
		result.AddFinalized(settlement.ID, settlement.Asset, settlement.Fee)
		l.Events.Add(l.Clock, "settlement.finalized", settlement.ID.String(), settlement.Asset, settlement.Amount, map[string]interface{}{
			"net_amount": settlement.NetAmount,
			"fee":        settlement.Fee,
		})
	}
	return result
}

func (l *Ledger) CloseActiveCycle() error {
	if l.ActiveCycle == nil {
		return ErrNoOpenCycle
	}
	cycle := l.ActiveCycle
	cycle.MarkDraining()
	for _, id := range cycle.Forecasts {
		forecast := l.Forecasts[id]
		if forecast == nil || forecast.Status == ForecastClosed {
			continue
		}
		source, err := l.Vault(forecast.SourceVault)
		if err != nil {
			return err
		}
		remainder := forecast.CloseRemainder(l.Clock)
		released := source.ReleaseForecast(remainder)
		cycle.AddRelease(forecast.Asset, released)
		if released > 0 {
			l.Events.Add(l.Clock, "forecast.closed_release", forecast.ID.String(), forecast.Asset, released, map[string]interface{}{
				"route": forecast.Route.String(),
			})
		}
	}
	cycle.Close(l.Clock)
	l.Events.Add(l.Clock, "cycle.closed", cycle.ID.String(), "", Amount(len(cycle.Forecasts)), nil)
	return nil
}

func (l *Ledger) Totals() TotalsReport {
	totals := TotalsReport{
		Reserves:          AmountBuckets{},
		FreeLiquidity:     AmountBuckets{},
		ForecastCommitted: AmountBuckets{},
		ReservationHeld:   AmountBuckets{},
		SettlementDebt:    AmountBuckets{},
		PendingIn:         AmountBuckets{},
		PendingOut:        AmountBuckets{},
		ReservedRoutes:    AmountBuckets{},
		SettledRoutes:     AmountBuckets{},
		ReleasedVariances: AmountBuckets{},
		WithdrawalDebits:  AmountBuckets{},
		FinalizedFees:     AmountBuckets{},
	}
	for _, id := range SortedVaultIDs(l.Vaults) {
		vault := l.Vaults[id]
		totals.Reserves.Add(vault.Asset, vault.Reserve)
		totals.FreeLiquidity.Add(vault.Asset, vault.FreeLiquidity())
		totals.ForecastCommitted.Add(vault.Asset, vault.ForecastCommitted)
		totals.ReservationHeld.Add(vault.Asset, vault.ReservationHeld)
		totals.SettlementDebt.Add(vault.Asset, vault.SettlementDebt)
		totals.PendingIn.Add(vault.Asset, vault.PendingIn)
		totals.PendingOut.Add(vault.Asset, vault.PendingOut)
	}
	for _, id := range SortedRouteIDs(l.Routes) {
		route := l.Routes[id]
		totals.ReservedRoutes.Add(route.Asset, route.Reserved)
		totals.SettledRoutes.Add(route.Asset, route.Settled)
		totals.ReleasedVariances.Add(route.Asset, route.ReleasedVariance)
	}
	for _, id := range SortedWithdrawalIDs(l.Withdrawals) {
		withdrawal := l.Withdrawals[id]
		if withdrawal.Status == WithdrawalCompleted {
			totals.WithdrawalDebits.Add(withdrawal.Asset, withdrawal.Amount)
		}
	}
	for _, id := range SortedSettlementIDs(l.Settlements) {
		settlement := l.Settlements[id]
		if settlement.Status == SettlementFinalized {
			totals.FinalizedFees.Add(settlement.Asset, settlement.Fee)
		}
	}
	return totals
}
