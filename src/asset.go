package main

type AssetKind string

const (
	AssetStable     AssetKind = "stable"
	AssetCollateral AssetKind = "collateral"
	AssetInternal   AssetKind = "internal"
)

type Asset struct {
	ID                 AssetID   `json:"id"`
	Symbol             string    `json:"symbol"`
	Decimals           int       `json:"decimals"`
	Kind               AssetKind `json:"kind"`
	ForecastHaircutBps Amount    `json:"forecast_haircut_bps"`
	SettlementFeeBps   Amount    `json:"settlement_fee_bps"`
	ReserveFloorBps    Amount    `json:"reserve_floor_bps"`
	RiskWeightBps      Amount    `json:"risk_weight_bps"`
}

func NewAsset(id AssetID, symbol string, decimals int, kind AssetKind) *Asset {
	return &Asset{
		ID:                 id,
		Symbol:             symbol,
		Decimals:           decimals,
		Kind:               kind,
		ForecastHaircutBps: 9_700,
		SettlementFeeBps:   6,
		ReserveFloorBps:    250,
		RiskWeightBps:      10_000,
	}
}

func (a *Asset) Clone() Asset {
	if a == nil {
		return Asset{}
	}
	return *a
}

func (a *Asset) Forecastable(amount Amount) Amount {
	if a == nil {
		return 0
	}
	return amount.MulBpsFloor(a.ForecastHaircutBps)
}

func (a *Asset) SettlementFee(amount Amount) Amount {
	if a == nil {
		return 0
	}
	return amount.MulBpsCeil(a.SettlementFeeBps)
}

func (a *Asset) FloorFor(reserve Amount) Amount {
	if a == nil {
		return 0
	}
	return reserve.MulBpsCeil(a.ReserveFloorBps)
}

func (a *Asset) RiskWeighted(amount Amount) Amount {
	if a == nil {
		return amount
	}
	return amount.MulBpsCeil(a.RiskWeightBps)
}

type AssetRegistry struct {
	items map[AssetID]*Asset
}

func NewAssetRegistry() *AssetRegistry {
	return &AssetRegistry{items: map[AssetID]*Asset{}}
}

func (r *AssetRegistry) Add(asset *Asset) {
	if asset == nil {
		return
	}
	copied := asset.Clone()
	r.items[asset.ID] = &copied
}

func (r *AssetRegistry) Get(id AssetID) (*Asset, error) {
	asset, ok := r.items[id]
	if !ok {
		return nil, ErrUnknownAsset
	}
	return asset, nil
}

func (r *AssetRegistry) MustGet(id AssetID) *Asset {
	asset, err := r.Get(id)
	if err != nil {
		panic(err)
	}
	return asset
}

func (r *AssetRegistry) Exists(id AssetID) bool {
	_, ok := r.items[id]
	return ok
}

func (r *AssetRegistry) Reports() []Asset {
	ids := SortedAssetIDs(r.items)
	reports := make([]Asset, 0, len(ids))
	for _, id := range ids {
		reports = append(reports, r.items[id].Clone())
	}
	return reports
}

func (r *AssetRegistry) Buckets() AmountBuckets {
	out := AmountBuckets{}
	for _, id := range SortedAssetIDs(r.items) {
		out.Add(id, 0)
	}
	return out
}
