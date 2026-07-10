package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

func EncodeJSON(value interface{}) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

func PrintJSON(value interface{}) error {
	data, err := EncodeJSON(value)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(data, '\n'))
	return err
}

func DigestReport(report OmenReport) string {
	type digestable struct {
		Lab           string
		Scenario      string
		NetworkID     string
		Clock         int
		Vaults        []VaultReport
		Routes        []RouteReport
		Forecasts     []ForecastEntry
		Reservations  []Reservation
		Settlements   []Settlement
		DemandUpdates []DemandUpdate
		Withdrawals   []Withdrawal
		Totals        TotalsReport
	}
	data, _ := json.Marshal(digestable{
		Lab:           report.Lab,
		Scenario:      report.Scenario,
		NetworkID:     report.NetworkID,
		Clock:         report.Clock,
		Vaults:        report.Vaults,
		Routes:        report.Routes,
		Forecasts:     report.Forecasts,
		Reservations:  report.Reservations,
		Settlements:   report.Settlements,
		DemandUpdates: report.DemandUpdates,
		Withdrawals:   report.Withdrawals,
		Totals:        report.Totals,
	})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:16])
}

func sortedCycleIDs(values map[CycleID]*ForecastCycle) []CycleID {
	ids := make([]CycleID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func ValidateReport(report OmenReport) error {
	if report.Lab != "OmenDTL" {
		return fmt.Errorf("%w: lab", ErrInvariantViolation)
	}
	inv := report.Invariants
	if !inv.VaultsNonNegative ||
		!inv.AccountsNonNegative ||
		!inv.RouteLinksValid ||
		!inv.ForecastLinksValid ||
		!inv.ReservationLinksValid ||
		!inv.SettlementLinksValid ||
		!inv.CycleLinksValid ||
		!inv.WithdrawalsAssetMatched ||
		!inv.ForecastAccountingNonNegative ||
		!inv.RoutesWithinForecastEnvelope {
		return ErrInvariantViolation
	}
	if report.StateDigest == "" {
		return fmt.Errorf("%w: digest", ErrInvariantViolation)
	}
	return nil
}
