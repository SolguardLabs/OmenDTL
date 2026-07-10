package main

type TotalsReport struct {
	Reserves          AmountBuckets `json:"reserves"`
	FreeLiquidity     AmountBuckets `json:"free_liquidity"`
	ForecastCommitted AmountBuckets `json:"forecast_committed"`
	ReservationHeld   AmountBuckets `json:"reservation_held"`
	SettlementDebt    AmountBuckets `json:"settlement_debt"`
	PendingIn         AmountBuckets `json:"pending_in"`
	PendingOut        AmountBuckets `json:"pending_out"`
	ReservedRoutes    AmountBuckets `json:"reserved_routes"`
	SettledRoutes     AmountBuckets `json:"settled_routes"`
	ReleasedVariances AmountBuckets `json:"released_variances"`
	WithdrawalDebits  AmountBuckets `json:"withdrawal_debits"`
	FinalizedFees     AmountBuckets `json:"finalized_fees"`
}

type OmenReport struct {
	Lab            string             `json:"lab"`
	Scenario       string             `json:"scenario"`
	NetworkID      string             `json:"network_id"`
	Clock          int                `json:"clock"`
	StateDigest    string             `json:"state_digest"`
	Assets         []Asset            `json:"assets"`
	Accounts       []AccountReport    `json:"accounts"`
	Vaults         []VaultReport      `json:"vaults"`
	Routes         []RouteReport      `json:"routes"`
	Cycles         []ForecastCycle    `json:"cycles"`
	Forecasts      []ForecastEntry    `json:"forecasts"`
	Reservations   []Reservation      `json:"reservations"`
	Settlements    []Settlement       `json:"settlements"`
	DemandUpdates  []DemandUpdate     `json:"demand_updates"`
	Withdrawals    []Withdrawal       `json:"withdrawals"`
	Views          []LiquidityView    `json:"views"`
	Batches        []ReservationBatch `json:"batches"`
	SettlementRuns []SettlementResult `json:"settlement_runs"`
	Totals         TotalsReport       `json:"totals"`
	Risk           RiskReport         `json:"risk"`
	Invariants     InvariantReport    `json:"invariants"`
	Events         []Event            `json:"events"`
	Notes          []string           `json:"notes"`
}

func BuildReport(scenario string, ledger *Ledger, views []LiquidityView, batches []ReservationBatch, settlementRuns []SettlementResult) OmenReport {
	report := OmenReport{
		Lab:            "OmenDTL",
		Scenario:       scenario,
		NetworkID:      ledger.NetworkID,
		Clock:          ledger.Clock,
		Assets:         ledger.Assets.Reports(),
		Accounts:       []AccountReport{},
		Vaults:         []VaultReport{},
		Routes:         []RouteReport{},
		Cycles:         []ForecastCycle{},
		Forecasts:      []ForecastEntry{},
		Reservations:   []Reservation{},
		Settlements:    []Settlement{},
		DemandUpdates:  []DemandUpdate{},
		Withdrawals:    []Withdrawal{},
		Views:          views,
		Batches:        batches,
		SettlementRuns: settlementRuns,
		Totals:         ledger.Totals(),
		Risk:           NewRiskEngine(ledger).Report(),
		Invariants:     NewRiskEngine(ledger).Invariants(),
		Events:         ledger.Events.All(),
		Notes:          append([]string{}, ledger.Notes...),
	}
	for _, id := range SortedAccountIDs(ledger.Accounts) {
		report.Accounts = append(report.Accounts, ledger.Accounts[id].Report())
	}
	for _, id := range SortedVaultIDs(ledger.Vaults) {
		report.Vaults = append(report.Vaults, ledger.Vaults[id].Report())
	}
	for _, id := range SortedRouteIDs(ledger.Routes) {
		report.Routes = append(report.Routes, ledger.Routes[id].Report())
	}
	for _, id := range sortedCycleIDs(ledger.Cycles) {
		report.Cycles = append(report.Cycles, *ledger.Cycles[id])
	}
	for _, id := range SortedForecastIDs(ledger.Forecasts) {
		report.Forecasts = append(report.Forecasts, *ledger.Forecasts[id])
	}
	for _, id := range SortedReservationIDs(ledger.Reservations) {
		report.Reservations = append(report.Reservations, *ledger.Reservations[id])
	}
	for _, id := range SortedSettlementIDs(ledger.Settlements) {
		report.Settlements = append(report.Settlements, *ledger.Settlements[id])
	}
	for _, id := range SortedDemandUpdateIDs(ledger.DemandUpdates) {
		report.DemandUpdates = append(report.DemandUpdates, ledger.DemandUpdates[id].Report())
	}
	for _, id := range SortedWithdrawalIDs(ledger.Withdrawals) {
		report.Withdrawals = append(report.Withdrawals, *ledger.Withdrawals[id])
	}
	report.StateDigest = DigestReport(report)
	return report
}
