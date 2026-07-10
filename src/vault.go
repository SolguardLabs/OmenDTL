package main

type VaultStatus string

const (
	VaultOpen     VaultStatus = "open"
	VaultWatching VaultStatus = "watching"
	VaultPaused   VaultStatus = "paused"
)

type Vault struct {
	ID                  VaultID     `json:"id"`
	Asset               AssetID     `json:"asset"`
	Region              string      `json:"region"`
	Reserve             Amount      `json:"reserve"`
	ForecastCommitted   Amount      `json:"forecast_committed"`
	ReservationHeld     Amount      `json:"reservation_held"`
	SettlementDebt      Amount      `json:"settlement_debt"`
	PendingIn           Amount      `json:"pending_in"`
	PendingOut          Amount      `json:"pending_out"`
	MinBuffer           Amount      `json:"min_buffer"`
	Priority            int         `json:"priority"`
	Strategy            string      `json:"strategy"`
	Status              VaultStatus `json:"status"`
	LastDemandClock     int         `json:"last_demand_clock"`
	LastSettlementClock int         `json:"last_settlement_clock"`
}

func NewVault(id VaultID, asset AssetID, region string, reserve Amount) *Vault {
	return &Vault{
		ID:       id,
		Asset:    asset,
		Region:   region,
		Reserve:  reserve,
		Status:   VaultOpen,
		Priority: 1,
		Strategy: "standard",
	}
}

func (v *Vault) Clone() Vault {
	if v == nil {
		return Vault{}
	}
	return *v
}

func (v *Vault) Exposure() Amount {
	return v.ForecastCommitted + v.ReservationHeld + v.SettlementDebt + v.PendingOut
}

func (v *Vault) LiquidReserve() Amount {
	if v.Reserve+v.PendingIn <= v.PendingOut {
		return 0
	}
	return v.Reserve + v.PendingIn - v.PendingOut
}

func (v *Vault) FreeLiquidity() Amount {
	liquid := v.LiquidReserve()
	locked := v.ForecastCommitted + v.ReservationHeld + v.SettlementDebt + v.MinBuffer
	if locked >= liquid {
		return 0
	}
	return liquid - locked
}

func (v *Vault) CoverageRatioBps() Amount {
	exposure := v.Exposure()
	if exposure == 0 {
		return BasisPoint
	}
	return PercentageOf(v.Reserve, exposure)
}

func (v *Vault) CommitForecast(amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if v.Status != VaultOpen {
		return ErrInsufficientLiquidity
	}
	if v.FreeLiquidity() < amount {
		return ErrInsufficientLiquidity
	}
	v.ForecastCommitted += amount
	return nil
}

func (v *Vault) ReleaseForecast(amount Amount) Amount {
	if amount <= 0 {
		return 0
	}
	released := MinAmount(v.ForecastCommitted, amount)
	v.ForecastCommitted -= released
	return released
}

func (v *Vault) AttachReservation(amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if amount == 0 {
		return nil
	}
	if v.ForecastCommitted >= amount {
		v.ForecastCommitted -= amount
		v.ReservationHeld += amount
		return nil
	}
	short := amount - v.ForecastCommitted
	if v.FreeLiquidity() < short {
		return ErrInsufficientLiquidity
	}
	v.ForecastCommitted = 0
	v.ReservationHeld += amount
	return nil
}

func (v *Vault) ReleaseReservation(amount Amount) Amount {
	if amount <= 0 {
		return 0
	}
	released := MinAmount(v.ReservationHeld, amount)
	v.ReservationHeld -= released
	return released
}

func (v *Vault) OpenSettlementDebt(amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if v.ReservationHeld >= amount {
		v.ReservationHeld -= amount
		v.SettlementDebt += amount
		return nil
	}
	short := amount - v.ReservationHeld
	if v.FreeLiquidity() < short {
		return ErrInsufficientLiquidity
	}
	v.ReservationHeld = 0
	v.SettlementDebt += amount
	return nil
}

func (v *Vault) CompleteSettlementOut(amount Amount, clock int) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if v.SettlementDebt < amount {
		return ErrAmountUnderflow
	}
	if v.Reserve < amount+v.MinBuffer {
		return ErrInsufficientLiquidity
	}
	v.SettlementDebt -= amount
	v.Reserve -= amount
	v.LastSettlementClock = clock
	return nil
}

func (v *Vault) ReceiveSettlement(amount Amount, clock int) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	v.Reserve += amount
	if v.PendingIn >= amount {
		v.PendingIn -= amount
	}
	v.LastSettlementClock = clock
	return nil
}

func (v *Vault) QueueInbound(amount Amount) {
	if amount > 0 {
		v.PendingIn += amount
	}
}

func (v *Vault) QueueOutbound(amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if v.FreeLiquidity() < amount {
		return ErrInsufficientLiquidity
	}
	v.PendingOut += amount
	return nil
}

func (v *Vault) CompleteOutbound(amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if v.PendingOut < amount {
		return ErrAmountUnderflow
	}
	if v.Reserve < amount {
		return ErrInsufficientLiquidity
	}
	v.PendingOut -= amount
	v.Reserve -= amount
	return nil
}

func (v *Vault) Withdraw(amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if v.FreeLiquidity() < amount {
		return ErrInsufficientLiquidity
	}
	if v.Reserve < amount {
		return ErrInsufficientLiquidity
	}
	v.Reserve -= amount
	return nil
}

func (v *Vault) Deposit(amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	v.Reserve += amount
	return nil
}

func (v *Vault) NonNegative() bool {
	return v.Reserve >= 0 &&
		v.ForecastCommitted >= 0 &&
		v.ReservationHeld >= 0 &&
		v.SettlementDebt >= 0 &&
		v.PendingIn >= 0 &&
		v.PendingOut >= 0 &&
		v.MinBuffer >= 0
}

type VaultReport struct {
	ID                  VaultID     `json:"id"`
	Asset               AssetID     `json:"asset"`
	Region              string      `json:"region"`
	Reserve             Amount      `json:"reserve"`
	ForecastCommitted   Amount      `json:"forecast_committed"`
	ReservationHeld     Amount      `json:"reservation_held"`
	SettlementDebt      Amount      `json:"settlement_debt"`
	PendingIn           Amount      `json:"pending_in"`
	PendingOut          Amount      `json:"pending_out"`
	MinBuffer           Amount      `json:"min_buffer"`
	FreeLiquidity       Amount      `json:"free_liquidity"`
	Exposure            Amount      `json:"exposure"`
	CoverageRatioBps    Amount      `json:"coverage_ratio_bps"`
	Priority            int         `json:"priority"`
	Strategy            string      `json:"strategy"`
	Status              VaultStatus `json:"status"`
	LastDemandClock     int         `json:"last_demand_clock"`
	LastSettlementClock int         `json:"last_settlement_clock"`
}

func (v *Vault) Report() VaultReport {
	return VaultReport{
		ID:                  v.ID,
		Asset:               v.Asset,
		Region:              v.Region,
		Reserve:             v.Reserve,
		ForecastCommitted:   v.ForecastCommitted,
		ReservationHeld:     v.ReservationHeld,
		SettlementDebt:      v.SettlementDebt,
		PendingIn:           v.PendingIn,
		PendingOut:          v.PendingOut,
		MinBuffer:           v.MinBuffer,
		FreeLiquidity:       v.FreeLiquidity(),
		Exposure:            v.Exposure(),
		CoverageRatioBps:    v.CoverageRatioBps(),
		Priority:            v.Priority,
		Strategy:            v.Strategy,
		Status:              v.Status,
		LastDemandClock:     v.LastDemandClock,
		LastSettlementClock: v.LastSettlementClock,
	}
}
