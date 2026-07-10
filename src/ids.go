package main

import (
	"fmt"
	"sort"
	"strings"
)

type AssetID string
type AccountID string
type VaultID string
type RouteID string
type ForecastID string
type ReservationID string
type SettlementID string
type CycleID string
type DemandUpdateID string
type WithdrawalID string

func (id AssetID) String() string       { return string(id) }
func (id AccountID) String() string     { return string(id) }
func (id VaultID) String() string       { return string(id) }
func (id RouteID) String() string       { return string(id) }
func (id ForecastID) String() string    { return string(id) }
func (id ReservationID) String() string { return string(id) }
func (id SettlementID) String() string  { return string(id) }
func (id CycleID) String() string       { return string(id) }
func (id DemandUpdateID) String() string {
	return string(id)
}
func (id WithdrawalID) String() string { return string(id) }

func NormalizeID(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		part = strings.ReplaceAll(part, "_", "-")
		part = strings.ReplaceAll(part, " ", "-")
		for strings.Contains(part, "--") {
			part = strings.ReplaceAll(part, "--", "-")
		}
		part = strings.Trim(part, "-")
		if part != "" {
			clean = append(clean, part)
		}
	}
	return strings.Join(clean, "-")
}

func NewVaultID(region string, asset AssetID) VaultID {
	return VaultID(NormalizeID("vault", region, asset.String()))
}

func NewRouteID(source VaultID, target VaultID) RouteID {
	return RouteID(NormalizeID("route", source.String(), target.String()))
}

func NewForecastID(cycle CycleID, route RouteID) ForecastID {
	return ForecastID(NormalizeID("forecast", cycle.String(), route.String()))
}

func NewReservationID(cycle CycleID, route RouteID, seq int) ReservationID {
	return ReservationID(NormalizeID("reserve", cycle.String(), route.String(), fmt.Sprintf("%03d", seq)))
}

func NewSettlementID(reservation ReservationID) SettlementID {
	return SettlementID(NormalizeID("settle", reservation.String()))
}

func NewDemandUpdateID(cycle CycleID, route RouteID, seq int) DemandUpdateID {
	return DemandUpdateID(NormalizeID("demand", cycle.String(), route.String(), fmt.Sprintf("%03d", seq)))
}

func NewWithdrawalID(vault VaultID, account AccountID, seq int) WithdrawalID {
	return WithdrawalID(NormalizeID("withdraw", vault.String(), account.String(), fmt.Sprintf("%03d", seq)))
}

func SortedAssetIDs(values map[AssetID]*Asset) []AssetID {
	ids := make([]AssetID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedAccountIDs(values map[AccountID]*Account) []AccountID {
	ids := make([]AccountID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedVaultIDs(values map[VaultID]*Vault) []VaultID {
	ids := make([]VaultID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedRouteIDs(values map[RouteID]*Route) []RouteID {
	ids := make([]RouteID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedForecastIDs(values map[ForecastID]*ForecastEntry) []ForecastID {
	ids := make([]ForecastID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedReservationIDs(values map[ReservationID]*Reservation) []ReservationID {
	ids := make([]ReservationID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedSettlementIDs(values map[SettlementID]*Settlement) []SettlementID {
	ids := make([]SettlementID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedDemandUpdateIDs(values map[DemandUpdateID]*DemandUpdate) []DemandUpdateID {
	ids := make([]DemandUpdateID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func SortedWithdrawalIDs(values map[WithdrawalID]*Withdrawal) []WithdrawalID {
	ids := make([]WithdrawalID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
