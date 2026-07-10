package main

type PlannerMode string

const (
	PlannerObserved  PlannerMode = "observed"
	PlannerProjected PlannerMode = "projected"
	PlannerDraining  PlannerMode = "draining"
)

type LiquidityProfile struct {
	Vault              VaultID     `json:"vault"`
	Asset              AssetID     `json:"asset"`
	Region             string      `json:"region"`
	Mode               PlannerMode `json:"mode"`
	Reserve            Amount      `json:"reserve"`
	FreeLiquidity      Amount      `json:"free_liquidity"`
	ForecastCommitted  Amount      `json:"forecast_committed"`
	ReservationHeld    Amount      `json:"reservation_held"`
	SettlementDebt     Amount      `json:"settlement_debt"`
	PendingIn          Amount      `json:"pending_in"`
	PendingOut         Amount      `json:"pending_out"`
	ProjectedInbound   Amount      `json:"projected_inbound"`
	ProjectedOutbound  Amount      `json:"projected_outbound"`
	ProjectedAvailable Amount      `json:"projected_available"`
	CoverageRatioBps   Amount      `json:"coverage_ratio_bps"`
	Priority           int         `json:"priority"`
	Status             VaultStatus `json:"status"`
}

type RouteProfile struct {
	Route             RouteID `json:"route"`
	Asset             AssetID `json:"asset"`
	Market            string  `json:"market"`
	SourceVault       VaultID `json:"source_vault"`
	TargetVault       VaultID `json:"target_vault"`
	OpenDemand        Amount  `json:"open_demand"`
	ForecastRemaining Amount  `json:"forecast_remaining"`
	Reserved          Amount  `json:"reserved"`
	Settled           Amount  `json:"settled"`
	Priority          int     `json:"priority"`
}

type LiquidityView struct {
	ID       string             `json:"id"`
	Clock    int                `json:"clock"`
	Mode     PlannerMode        `json:"mode"`
	Profiles []LiquidityProfile `json:"profiles"`
	Routes   []RouteProfile     `json:"routes"`
	Totals   AmountBuckets      `json:"totals"`
	Warnings []string           `json:"warnings"`
}

type Planner struct {
	Ledger *Ledger
}

func NewPlanner(ledger *Ledger) *Planner {
	return &Planner{Ledger: ledger}
}

func (p *Planner) Build(mode PlannerMode) LiquidityView {
	view := LiquidityView{
		ID:       NormalizeID("view", string(mode), p.Ledger.ClockString()),
		Clock:    p.Ledger.Clock,
		Mode:     mode,
		Profiles: []LiquidityProfile{},
		Routes:   []RouteProfile{},
		Totals:   AmountBuckets{},
		Warnings: []string{},
	}
	for _, id := range SortedVaultIDs(p.Ledger.Vaults) {
		vault := p.Ledger.Vaults[id]
		profile := p.profile(vault, mode)
		view.Profiles = append(view.Profiles, profile)
		view.Totals.Add(vault.Asset, profile.ProjectedAvailable)
		if profile.CoverageRatioBps < 9_000 {
			view.Warnings = append(view.Warnings, NormalizeID("coverage", vault.ID.String(), profile.CoverageRatioBps.String()))
		}
	}
	for _, id := range SortedRouteIDs(p.Ledger.Routes) {
		route := p.Ledger.Routes[id]
		view.Routes = append(view.Routes, RouteProfile{
			Route:             route.ID,
			Asset:             route.Asset,
			Market:            route.Market,
			SourceVault:       route.SourceVault,
			TargetVault:       route.TargetVault,
			OpenDemand:        route.OpenDemand(),
			ForecastRemaining: route.ForecastRemaining(),
			Reserved:          route.Reserved,
			Settled:           route.Settled,
			Priority:          route.Priority,
		})
	}
	return view
}

func (p *Planner) profile(vault *Vault, mode PlannerMode) LiquidityProfile {
	projectedInbound := vault.PendingIn
	projectedOutbound := vault.PendingOut
	if mode == PlannerProjected && p.Ledger.ActiveCycle != nil {
		for _, id := range p.Ledger.ActiveCycle.Forecasts {
			forecast := p.Ledger.Forecasts[id]
			if forecast.TargetVault == vault.ID {
				projectedInbound += forecast.ReservedAmount
			}
			if forecast.SourceVault == vault.ID {
				projectedOutbound += forecast.ReservedAmount
			}
		}
	}
	if mode == PlannerDraining {
		for _, settlement := range p.Ledger.Settlements {
			if settlement.Status != SettlementPending {
				continue
			}
			if settlement.TargetVault == vault.ID {
				projectedInbound += settlement.NetAmount
			}
			if settlement.SourceVault == vault.ID {
				projectedOutbound += settlement.Amount
			}
		}
	}
	available := vault.Reserve + projectedInbound
	locked := vault.ForecastCommitted + vault.ReservationHeld + vault.SettlementDebt + projectedOutbound + vault.MinBuffer
	if locked >= available {
		available = 0
	} else {
		available -= locked
	}
	return LiquidityProfile{
		Vault:              vault.ID,
		Asset:              vault.Asset,
		Region:             vault.Region,
		Mode:               mode,
		Reserve:            vault.Reserve,
		FreeLiquidity:      vault.FreeLiquidity(),
		ForecastCommitted:  vault.ForecastCommitted,
		ReservationHeld:    vault.ReservationHeld,
		SettlementDebt:     vault.SettlementDebt,
		PendingIn:          vault.PendingIn,
		PendingOut:         vault.PendingOut,
		ProjectedInbound:   projectedInbound,
		ProjectedOutbound:  projectedOutbound,
		ProjectedAvailable: available,
		CoverageRatioBps:   vault.CoverageRatioBps(),
		Priority:           vault.Priority,
		Status:             vault.Status,
	}
}

func (p *Planner) SuggestWithdrawals(asset AssetID, floor Amount) []WithdrawalPlan {
	plans := []WithdrawalPlan{}
	for _, id := range SortedVaultIDs(p.Ledger.Vaults) {
		vault := p.Ledger.Vaults[id]
		if vault.Asset != asset {
			continue
		}
		free := vault.FreeLiquidity()
		if free <= floor {
			continue
		}
		amount := free - floor
		plans = append(plans, WithdrawalPlan{
			Vault:  vault.ID,
			Asset:  asset,
			Amount: amount,
			Reason: "liquidity above policy floor",
		})
	}
	return plans
}

type WithdrawalPlan struct {
	Vault  VaultID `json:"vault"`
	Asset  AssetID `json:"asset"`
	Amount Amount  `json:"amount"`
	Reason string  `json:"reason"`
}
