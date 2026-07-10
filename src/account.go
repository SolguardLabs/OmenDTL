package main

type AccountRole string

const (
	RoleProtocol  AccountRole = "protocol"
	RoleOperator  AccountRole = "operator"
	RoleAllocator AccountRole = "allocator"
	RoleTreasury  AccountRole = "treasury"
	RoleCustomer  AccountRole = "customer"
)

type Account struct {
	ID       AccountID          `json:"id"`
	Role     AccountRole        `json:"role"`
	Balances map[AssetID]Amount `json:"balances"`
	Holds    map[AssetID]Amount `json:"holds"`
	Nonce    int                `json:"nonce"`
	Active   bool               `json:"active"`
}

func NewAccount(id AccountID, role AccountRole) *Account {
	return &Account{
		ID:       id,
		Role:     role,
		Balances: map[AssetID]Amount{},
		Holds:    map[AssetID]Amount{},
		Active:   true,
	}
}

func (a *Account) Clone() Account {
	copied := Account{
		ID:       a.ID,
		Role:     a.Role,
		Balances: map[AssetID]Amount{},
		Holds:    map[AssetID]Amount{},
		Nonce:    a.Nonce,
		Active:   a.Active,
	}
	for asset, amount := range a.Balances {
		copied.Balances[asset] = amount
	}
	for asset, amount := range a.Holds {
		copied.Holds[asset] = amount
	}
	return copied
}

func (a *Account) Balance(asset AssetID) Amount {
	if a == nil {
		return 0
	}
	return a.Balances[asset]
}

func (a *Account) Hold(asset AssetID) Amount {
	if a == nil {
		return 0
	}
	return a.Holds[asset]
}

func (a *Account) Available(asset AssetID) Amount {
	balance := a.Balance(asset)
	hold := a.Hold(asset)
	if hold >= balance {
		return 0
	}
	return balance - hold
}

func (a *Account) Deposit(asset AssetID, amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	a.Balances[asset] += amount
	a.Nonce++
	return nil
}

func (a *Account) Debit(asset AssetID, amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if a.Available(asset) < amount {
		return ErrInsufficientBalance
	}
	a.Balances[asset] -= amount
	a.Nonce++
	return nil
}

func (a *Account) PlaceHold(asset AssetID, amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if a.Available(asset) < amount {
		return ErrInsufficientBalance
	}
	a.Holds[asset] += amount
	a.Nonce++
	return nil
}

func (a *Account) ReleaseHold(asset AssetID, amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if a.Holds[asset] < amount {
		return ErrAmountUnderflow
	}
	a.Holds[asset] -= amount
	a.Nonce++
	return nil
}

func (a *Account) CaptureHold(asset AssetID, amount Amount) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	if a.Holds[asset] < amount || a.Balances[asset] < amount {
		return ErrInsufficientBalance
	}
	a.Holds[asset] -= amount
	a.Balances[asset] -= amount
	a.Nonce++
	return nil
}

func (a *Account) NonNegative() bool {
	for _, balance := range a.Balances {
		if balance < 0 {
			return false
		}
	}
	for _, hold := range a.Holds {
		if hold < 0 {
			return false
		}
	}
	return true
}

type AccountReport struct {
	ID        AccountID     `json:"id"`
	Role      AccountRole   `json:"role"`
	Balances  AmountBuckets `json:"balances"`
	Holds     AmountBuckets `json:"holds"`
	Available AmountBuckets `json:"available"`
	Nonce     int           `json:"nonce"`
	Active    bool          `json:"active"`
}

func (a *Account) Report() AccountReport {
	balances := AmountBuckets{}
	holds := AmountBuckets{}
	available := AmountBuckets{}
	for asset, amount := range a.Balances {
		balances.Add(asset, amount)
		available.Add(asset, a.Available(asset))
	}
	for asset, amount := range a.Holds {
		holds.Add(asset, amount)
	}
	return AccountReport{
		ID:        a.ID,
		Role:      a.Role,
		Balances:  balances,
		Holds:     holds,
		Available: available,
		Nonce:     a.Nonce,
		Active:    a.Active,
	}
}
