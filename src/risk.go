package main

type RiskReport struct {
	OpenCycles             int    `json:"open_cycles"`
	OpenForecasts          int    `json:"open_forecasts"`
	QueuedReservations     int    `json:"queued_reservations"`
	BoundReservations      int    `json:"bound_reservations"`
	PendingSettlements     int    `json:"pending_settlements"`
	FinalizedSettlements   int    `json:"finalized_settlements"`
	FrozenSettlements      int    `json:"frozen_settlements"`
	RejectedWithdrawals    int    `json:"rejected_withdrawals"`
	CompletedWithdrawals   int    `json:"completed_withdrawals"`
	ForecastCoverage       Amount `json:"forecast_coverage_bps"`
	ReserveCoverage        Amount `json:"reserve_coverage_bps"`
	ReleasedVariance       Amount `json:"released_variance"`
	LargestVaultFree       Amount `json:"largest_vault_free"`
	LargestRouteOpenDemand Amount `json:"largest_route_open_demand"`
}

type InvariantReport struct {
	VaultsNonNegative             bool `json:"vaults_non_negative"`
	AccountsNonNegative           bool `json:"accounts_non_negative"`
	RouteLinksValid               bool `json:"route_links_valid"`
	ForecastLinksValid            bool `json:"forecast_links_valid"`
	ReservationLinksValid         bool `json:"reservation_links_valid"`
	SettlementLinksValid          bool `json:"settlement_links_valid"`
	CycleLinksValid               bool `json:"cycle_links_valid"`
	WithdrawalsAssetMatched       bool `json:"withdrawals_asset_matched"`
	ForecastAccountingNonNegative bool `json:"forecast_accounting_non_negative"`
	RoutesWithinForecastEnvelope  bool `json:"routes_within_forecast_envelope"`
}

type RiskEngine struct {
	Ledger *Ledger
}

func NewRiskEngine(ledger *Ledger) *RiskEngine {
	return &RiskEngine{Ledger: ledger}
}

func (r *RiskEngine) Report() RiskReport {
	report := RiskReport{}
	var forecastActive Amount
	var forecastTotal Amount
	var reserves Amount
	var exposure Amount
	for _, cycle := range r.Ledger.Cycles {
		if cycle.Status != CycleClosed {
			report.OpenCycles++
		}
	}
	for _, forecast := range r.Ledger.Forecasts {
		forecastTotal += forecast.ForecastAmount
		if forecast.Status != ForecastClosed {
			report.OpenForecasts++
			forecastActive += forecast.ActiveBacking()
		}
		report.ReleasedVariance += forecast.ReleasedAmount
	}
	for _, reservation := range r.Ledger.Reservations {
		switch reservation.Status {
		case ReservationQueued:
			report.QueuedReservations++
		case ReservationBound, ReservationSettling:
			report.BoundReservations++
		}
	}
	for _, settlement := range r.Ledger.Settlements {
		switch settlement.Status {
		case SettlementPending:
			report.PendingSettlements++
		case SettlementFinalized:
			report.FinalizedSettlements++
		case SettlementFrozen:
			report.FrozenSettlements++
		}
	}
	for _, withdrawal := range r.Ledger.Withdrawals {
		switch withdrawal.Status {
		case WithdrawalCompleted:
			report.CompletedWithdrawals++
		case WithdrawalRejected:
			report.RejectedWithdrawals++
		}
	}
	for _, vault := range r.Ledger.Vaults {
		reserves += vault.Reserve
		exposure += vault.Exposure()
		report.LargestVaultFree = MaxAmount(report.LargestVaultFree, vault.FreeLiquidity())
	}
	for _, route := range r.Ledger.Routes {
		report.LargestRouteOpenDemand = MaxAmount(report.LargestRouteOpenDemand, route.OpenDemand())
	}
	report.ForecastCoverage = PercentageOf(forecastActive, MaxAmount(forecastTotal, 1))
	report.ReserveCoverage = PercentageOf(reserves, MaxAmount(exposure, 1))
	return report
}

func (r *RiskEngine) Invariants() InvariantReport {
	return InvariantReport{
		VaultsNonNegative:             r.vaultsNonNegative(),
		AccountsNonNegative:           r.accountsNonNegative(),
		RouteLinksValid:               r.routeLinksValid(),
		ForecastLinksValid:            r.forecastLinksValid(),
		ReservationLinksValid:         r.reservationLinksValid(),
		SettlementLinksValid:          r.settlementLinksValid(),
		CycleLinksValid:               r.cycleLinksValid(),
		WithdrawalsAssetMatched:       r.withdrawalsAssetMatched(),
		ForecastAccountingNonNegative: r.forecastAccountingNonNegative(),
		RoutesWithinForecastEnvelope:  r.routesWithinEnvelope(),
	}
}

func (r *RiskEngine) AllPassed() bool {
	inv := r.Invariants()
	return inv.VaultsNonNegative &&
		inv.AccountsNonNegative &&
		inv.RouteLinksValid &&
		inv.ForecastLinksValid &&
		inv.ReservationLinksValid &&
		inv.SettlementLinksValid &&
		inv.CycleLinksValid &&
		inv.WithdrawalsAssetMatched &&
		inv.ForecastAccountingNonNegative &&
		inv.RoutesWithinForecastEnvelope
}

func (r *RiskEngine) vaultsNonNegative() bool {
	for _, vault := range r.Ledger.Vaults {
		if !vault.NonNegative() {
			return false
		}
	}
	return true
}

func (r *RiskEngine) accountsNonNegative() bool {
	for _, account := range r.Ledger.Accounts {
		if !account.NonNegative() {
			return false
		}
	}
	return true
}

func (r *RiskEngine) routeLinksValid() bool {
	for _, route := range r.Ledger.Routes {
		source, sourceOK := r.Ledger.Vaults[route.SourceVault]
		target, targetOK := r.Ledger.Vaults[route.TargetVault]
		if !sourceOK || !targetOK {
			return false
		}
		if source.Asset != route.Asset || target.Asset != route.Asset {
			return false
		}
	}
	return true
}

func (r *RiskEngine) forecastLinksValid() bool {
	for _, forecast := range r.Ledger.Forecasts {
		route, routeOK := r.Ledger.Routes[forecast.Route]
		source, sourceOK := r.Ledger.Vaults[forecast.SourceVault]
		target, targetOK := r.Ledger.Vaults[forecast.TargetVault]
		if !routeOK || !sourceOK || !targetOK {
			return false
		}
		if route.SourceVault != forecast.SourceVault || route.TargetVault != forecast.TargetVault {
			return false
		}
		if source.Asset != forecast.Asset || target.Asset != forecast.Asset {
			return false
		}
	}
	return true
}

func (r *RiskEngine) reservationLinksValid() bool {
	for _, reservation := range r.Ledger.Reservations {
		if _, ok := r.Ledger.Forecasts[reservation.Forecast]; !ok {
			return false
		}
		if _, ok := r.Ledger.Routes[reservation.Route]; !ok {
			return false
		}
		if _, ok := r.Ledger.Vaults[reservation.SourceVault]; !ok {
			return false
		}
		if _, ok := r.Ledger.Vaults[reservation.TargetVault]; !ok {
			return false
		}
		if _, ok := r.Ledger.Accounts[reservation.Owner]; !ok {
			return false
		}
	}
	return true
}

func (r *RiskEngine) settlementLinksValid() bool {
	for _, settlement := range r.Ledger.Settlements {
		if _, ok := r.Ledger.Reservations[settlement.Reservation]; !ok {
			return false
		}
		if _, ok := r.Ledger.Routes[settlement.Route]; !ok {
			return false
		}
		if _, ok := r.Ledger.Vaults[settlement.SourceVault]; !ok {
			return false
		}
		if _, ok := r.Ledger.Vaults[settlement.TargetVault]; !ok {
			return false
		}
	}
	return true
}

func (r *RiskEngine) cycleLinksValid() bool {
	for _, cycle := range r.Ledger.Cycles {
		for _, forecastID := range cycle.Forecasts {
			forecast, ok := r.Ledger.Forecasts[forecastID]
			if !ok {
				return false
			}
			if forecast.Cycle != cycle.ID {
				return false
			}
		}
	}
	return true
}

func (r *RiskEngine) withdrawalsAssetMatched() bool {
	for _, withdrawal := range r.Ledger.Withdrawals {
		vault, ok := r.Ledger.Vaults[withdrawal.Vault]
		if !ok {
			return false
		}
		if vault.Asset != withdrawal.Asset {
			return false
		}
	}
	return true
}

func (r *RiskEngine) forecastAccountingNonNegative() bool {
	for _, forecast := range r.Ledger.Forecasts {
		if forecast.ForecastAmount < 0 ||
			forecast.CommittedAmount < 0 ||
			forecast.ReservedAmount < 0 ||
			forecast.SettledAmount < 0 ||
			forecast.ReleasedAmount < 0 ||
			forecast.FailedAmount < 0 {
			return false
		}
	}
	return true
}

func (r *RiskEngine) routesWithinEnvelope() bool {
	for _, route := range r.Ledger.Routes {
		if route.Settled+route.Failed > route.ForecastedDemand+route.ReleasedVariance+route.Policy.MaxReservation {
			return false
		}
	}
	return true
}
