package main

import "fmt"

type ScenarioRun struct {
	Name           string
	Ledger         *Ledger
	Views          []LiquidityView
	Batches        []ReservationBatch
	SettlementRuns []SettlementResult
}

func AvailableScenarios() []string {
	return []string{
		"baseline",
		"forecast-cycle",
		"reservation",
		"demand-update",
		"settlement",
		"liquidity-window",
		"operator-day",
	}
}

func RunScenario(name string) (*ScenarioRun, error) {
	switch name {
	case "baseline":
		return scenarioBaseline(), nil
	case "forecast-cycle":
		return scenarioForecastCycle(), nil
	case "reservation":
		return scenarioReservation(), nil
	case "demand-update":
		return scenarioDemandUpdate(), nil
	case "settlement":
		return scenarioSettlement(), nil
	case "liquidity-window":
		return scenarioLiquidityWindow(), nil
	case "operator-day":
		return scenarioOperatorDay(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidScenario, name)
	}
}

func NewScenarioRun(name string, ledger *Ledger) *ScenarioRun {
	return &ScenarioRun{
		Name:           name,
		Ledger:         ledger,
		Views:          []LiquidityView{},
		Batches:        []ReservationBatch{},
		SettlementRuns: []SettlementResult{},
	}
}

func (s *ScenarioRun) AddView(mode PlannerMode) {
	s.Views = append(s.Views, NewPlanner(s.Ledger).Build(mode))
}

func (s *ScenarioRun) AddBatch(batch *ReservationBatch) {
	if batch == nil {
		return
	}
	s.Batches = append(s.Batches, *batch)
}

func (s *ScenarioRun) AddSettlement(result SettlementResult) {
	s.SettlementRuns = append(s.SettlementRuns, result)
}

func (s *ScenarioRun) Report() OmenReport {
	return BuildReport(s.Name, s.Ledger, s.Views, s.Batches, s.SettlementRuns)
}

func BuildSeedLedger() *Ledger {
	ledger := NewLedger("omen-local-liquidity")
	usdc := NewAsset("ousdc", "oUSDC", 6, AssetStable)
	usdc.ForecastHaircutBps = 9_750
	usdc.SettlementFeeBps = 5
	usdc.ReserveFloorBps = 350
	eurc := NewAsset("oeur", "oEUR", 6, AssetStable)
	eurc.ForecastHaircutBps = 9_650
	eurc.SettlementFeeBps = 7
	eurc.ReserveFloorBps = 400
	guard := NewAsset("oguard", "oGUARD", 8, AssetCollateral)
	guard.ForecastHaircutBps = 8_800
	guard.RiskWeightBps = 7_500
	ledger.AddAsset(usdc)
	ledger.AddAsset(eurc)
	ledger.AddAsset(guard)

	protocol := NewAccount("protocol", RoleProtocol)
	_ = protocol.Deposit("ousdc", 500_000)
	_ = protocol.Deposit("oeur", 250_000)
	operator := NewAccount("operator-a", RoleOperator)
	_ = operator.Deposit("ousdc", 220_000)
	allocator := NewAccount("allocator-prime", RoleAllocator)
	_ = allocator.Deposit("ousdc", 75_000)
	treasury := NewAccount("treasury", RoleTreasury)
	_ = treasury.Deposit("ousdc", 1_000_000)
	analyst := NewAccount("forecast-ops", RoleOperator)
	_ = analyst.Deposit("oeur", 120_000)
	ledger.AddAccount(protocol)
	ledger.AddAccount(operator)
	ledger.AddAccount(allocator)
	ledger.AddAccount(treasury)
	ledger.AddAccount(analyst)

	atlas := NewVault("vault-atlas-ousdc", "ousdc", "atlas", 5_200_000)
	atlas.MinBuffer = 320_000
	atlas.Priority = 9
	atlas.Strategy = "core-source"
	boreal := NewVault("vault-boreal-ousdc", "ousdc", "boreal", 1_150_000)
	boreal.MinBuffer = 160_000
	boreal.Priority = 8
	boreal.Strategy = "north-demand"
	cinder := NewVault("vault-cinder-ousdc", "ousdc", "cinder", 820_000)
	cinder.MinBuffer = 125_000
	cinder.Priority = 6
	cinder.Strategy = "merchant-edge"
	delta := NewVault("vault-delta-ousdc", "ousdc", "delta", 640_000)
	delta.MinBuffer = 90_000
	delta.Priority = 5
	delta.Strategy = "payout-edge"
	euroCore := NewVault("vault-euro-oeur", "oeur", "euro", 2_700_000)
	euroCore.MinBuffer = 210_000
	euroCore.Priority = 7
	euroCore.Strategy = "fx-source"
	iberia := NewVault("vault-iberia-oeur", "oeur", "iberia", 760_000)
	iberia.MinBuffer = 90_000
	iberia.Priority = 8
	iberia.Strategy = "merchant-eu"
	guardVault := NewVault("vault-guard-oguard", "oguard", "guard", 12_500_000)
	guardVault.MinBuffer = 2_000_000
	guardVault.Priority = 2
	guardVault.Strategy = "collateral-buffer"
	_ = ledger.AddVault(atlas)
	_ = ledger.AddVault(boreal)
	_ = ledger.AddVault(cinder)
	_ = ledger.AddVault(delta)
	_ = ledger.AddVault(euroCore)
	_ = ledger.AddVault(iberia)
	_ = ledger.AddVault(guardVault)

	alpha := NewRoute("route-atlas-boreal", "vault-atlas-ousdc", "vault-boreal-ousdc", "ousdc", "core-north", 900_000)
	alpha.Priority = 9
	alpha.Policy.MinReservation = 80_000
	alpha.Policy.MaxReservation = 1_100_000
	alpha.Policy.SettlementDelay = 3
	alpha.Tags["book"] = "card-settlement"
	alpha.Tags["tier"] = "prime"
	beta := NewRoute("route-atlas-cinder", "vault-atlas-ousdc", "vault-cinder-ousdc", "ousdc", "core-merchant", 650_000)
	beta.Priority = 7
	beta.Policy.MinReservation = 60_000
	beta.Policy.MaxReservation = 850_000
	beta.Policy.SettlementDelay = 4
	beta.Tags["book"] = "merchant-payout"
	gamma := NewRoute("route-boreal-delta", "vault-boreal-ousdc", "vault-delta-ousdc", "ousdc", "north-payout", 470_000)
	gamma.Priority = 8
	gamma.Policy.MinReservation = 70_000
	gamma.Policy.MaxReservation = 650_000
	gamma.Policy.SettlementDelay = 3
	gamma.Tags["book"] = "instant-payout"
	euro := NewRoute("route-euro-iberia", "vault-euro-oeur", "vault-iberia-oeur", "oeur", "eu-merchant", 540_000)
	euro.Priority = 8
	euro.Policy.MinReservation = 70_000
	euro.Policy.MaxReservation = 700_000
	euro.Policy.SettlementDelay = 4
	euro.Tags["book"] = "eur-payout"
	_ = ledger.AddRoute(alpha)
	_ = ledger.AddRoute(beta)
	_ = ledger.AddRoute(gamma)
	_ = ledger.AddRoute(euro)

	ledger.AddNote("seeded forecast-driven liquidity mesh with deterministic vault and route balances")
	return ledger
}

func scenarioBaseline() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("baseline", ledger)
	run.AddView(PlannerObserved)
	ledger.AddNote("baseline exposes observed vault capacity without opening a forecast window")
	return run
}

func scenarioForecastCycle() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("forecast-cycle", ledger)
	run.AddView(PlannerObserved)
	_, _ = ledger.OpenForecastCycle(6)
	run.AddView(PlannerProjected)
	ledger.AddNote("forecast cycle committed expected route liquidity for the active window")
	return run
}

func scenarioReservation() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("reservation", ledger)
	_, _ = ledger.OpenForecastCycle(6)
	batch := NewAllocator(ledger).ExecuteMode(AllocationBalanced, "allocator-prime")
	run.AddBatch(batch)
	run.AddView(PlannerProjected)
	ledger.AddNote("reservation batch assigned forecast capacity to demand routes")
	return run
}

func scenarioDemandUpdate() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("demand-update", ledger)
	_, _ = ledger.OpenForecastCycle(6)
	batch := NewAllocator(ledger).ExecuteMode(AllocationConservative, "allocator-prime")
	run.AddBatch(batch)
	ledger.Advance(1)
	updater := NewDemandUpdater(ledger)
	_, _ = updater.Apply(NewDemandSignal("route-atlas-boreal", DemandSourceMerchant, 520_000, 8_800, ledger.Clock))
	_, _ = updater.Apply(NewDemandSignal("route-boreal-delta", DemandSourceOracle, 510_000, 9_100, ledger.Clock))
	run.AddView(PlannerProjected)
	ledger.AddNote("demand updates refreshed route forecasts inside the active cycle")
	return run
}

func scenarioSettlement() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("settlement", ledger)
	_, _ = ledger.OpenForecastCycle(6)
	batch := NewAllocator(ledger).ExecuteMode(AllocationBalanced, "allocator-prime")
	run.AddBatch(batch)
	ledger.Advance(3)
	run.AddSettlement(ledger.FinalizeReadySettlements())
	ledger.Advance(1)
	run.AddSettlement(ledger.FinalizeReadySettlements())
	run.AddView(PlannerDraining)
	_ = ledger.CloseActiveCycle()
	ledger.AddNote("settlement finalized reservations that reached their route delay")
	return run
}

func scenarioLiquidityWindow() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("liquidity-window", ledger)
	_, _ = ledger.OpenForecastCycle(6)
	batch := NewAllocator(ledger).ExecuteMode(AllocationConservative, "allocator-prime")
	run.AddBatch(batch)
	ledger.Advance(1)
	updater := NewDemandUpdater(ledger)
	_, _ = updater.Apply(NewDemandSignal("route-atlas-cinder", DemandSourceMerchant, 520_000, 8_900, ledger.Clock))
	plans := NewPlanner(ledger).SuggestWithdrawals("ousdc", 1_250_000)
	if len(plans) > 0 {
		_, _ = ledger.WithdrawFreeLiquidity(plans[0].Vault, "treasury", MinAmount(plans[0].Amount, 180_000))
	}
	run.AddView(PlannerProjected)
	ledger.Advance(4)
	run.AddSettlement(ledger.FinalizeReadySettlements())
	ledger.AddNote("liquidity window composed demand refresh, policy withdrawal and settlement")
	return run
}

func scenarioOperatorDay() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("operator-day", ledger)
	run.AddView(PlannerObserved)
	_, _ = ledger.OpenForecastCycle(8)
	batch := NewAllocator(ledger).ExecuteMode(AllocationPriority, "allocator-prime")
	run.AddBatch(batch)
	run.AddView(PlannerProjected)
	ledger.Advance(1)
	updater := NewDemandUpdater(ledger)
	_, _ = updater.Apply(NewDemandSignal("route-atlas-boreal", DemandSourceMerchant, 820_000, 9_200, ledger.Clock))
	_, _ = updater.Apply(NewDemandSignal("route-euro-iberia", DemandSourceOracle, 610_000, 9_000, ledger.Clock))
	run.AddView(PlannerProjected)
	ledger.Advance(2)
	run.AddSettlement(ledger.FinalizeReadySettlements())
	ledger.Advance(2)
	run.AddSettlement(ledger.FinalizeReadySettlements())
	run.AddView(PlannerDraining)
	_ = ledger.CloseActiveCycle()
	ledger.AddNote("operator day ran forecast, route assignment, demand refresh and settlement passes")
	return run
}
