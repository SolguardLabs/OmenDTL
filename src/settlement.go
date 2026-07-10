package main

type SettlementStatus string

const (
	SettlementPending   SettlementStatus = "pending"
	SettlementFinalized SettlementStatus = "finalized"
	SettlementFrozen    SettlementStatus = "frozen"
	SettlementRejected  SettlementStatus = "rejected"
)

type Settlement struct {
	ID             SettlementID     `json:"id"`
	Cycle          CycleID          `json:"cycle"`
	Reservation    ReservationID    `json:"reservation"`
	Route          RouteID          `json:"route"`
	SourceVault    VaultID          `json:"source_vault"`
	TargetVault    VaultID          `json:"target_vault"`
	Asset          AssetID          `json:"asset"`
	Amount         Amount           `json:"amount"`
	Fee            Amount           `json:"fee"`
	NetAmount      Amount           `json:"net_amount"`
	DueClock       int              `json:"due_clock"`
	CreatedClock   int              `json:"created_clock"`
	FinalizedClock int              `json:"finalized_clock,omitempty"`
	Status         SettlementStatus `json:"status"`
	FailureReason  string           `json:"failure_reason,omitempty"`
}

func NewSettlement(reservation *Reservation, clock int) *Settlement {
	net := reservation.Amount
	if reservation.Fee < reservation.Amount {
		net = reservation.Amount - reservation.Fee
	}
	return &Settlement{
		ID:           NewSettlementID(reservation.ID),
		Cycle:        reservation.Cycle,
		Reservation:  reservation.ID,
		Route:        reservation.Route,
		SourceVault:  reservation.SourceVault,
		TargetVault:  reservation.TargetVault,
		Asset:        reservation.Asset,
		Amount:       reservation.Amount,
		Fee:          reservation.Fee,
		NetAmount:    net,
		DueClock:     reservation.SettlementDue,
		CreatedClock: clock,
		Status:       SettlementPending,
	}
}

func (s *Settlement) Ready(clock int) bool {
	return s.Status == SettlementPending && clock >= s.DueClock
}

func (s *Settlement) Finalize(clock int) {
	s.Status = SettlementFinalized
	s.FinalizedClock = clock
}

func (s *Settlement) Freeze(clock int, reason string) {
	s.Status = SettlementFrozen
	s.FinalizedClock = clock
	s.FailureReason = reason
}

func (s *Settlement) Reject(clock int, reason string) {
	s.Status = SettlementRejected
	s.FinalizedClock = clock
	s.FailureReason = reason
}

type SettlementResult struct {
	Finalized []SettlementID `json:"finalized"`
	Frozen    []SettlementID `json:"frozen"`
	Rejected  []SettlementID `json:"rejected"`
	Fees      AmountBuckets  `json:"fees"`
	Clock     int            `json:"clock"`
}

func NewSettlementResult(clock int) SettlementResult {
	return SettlementResult{
		Finalized: []SettlementID{},
		Frozen:    []SettlementID{},
		Rejected:  []SettlementID{},
		Fees:      AmountBuckets{},
		Clock:     clock,
	}
}

func (r *SettlementResult) AddFinalized(id SettlementID, asset AssetID, fee Amount) {
	r.Finalized = append(r.Finalized, id)
	r.Fees.Add(asset, fee)
}

func (r *SettlementResult) AddFrozen(id SettlementID) {
	r.Frozen = append(r.Frozen, id)
}

func (r *SettlementResult) AddRejected(id SettlementID) {
	r.Rejected = append(r.Rejected, id)
}
