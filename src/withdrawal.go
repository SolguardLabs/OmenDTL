package main

type WithdrawalStatus string

const (
	WithdrawalQueued    WithdrawalStatus = "queued"
	WithdrawalCompleted WithdrawalStatus = "completed"
	WithdrawalRejected  WithdrawalStatus = "rejected"
)

type Withdrawal struct {
	ID         WithdrawalID     `json:"id"`
	Vault      VaultID          `json:"vault"`
	Account    AccountID        `json:"account"`
	Asset      AssetID          `json:"asset"`
	Amount     Amount           `json:"amount"`
	FreeBefore Amount           `json:"free_before"`
	FreeAfter  Amount           `json:"free_after"`
	Clock      int              `json:"clock"`
	Status     WithdrawalStatus `json:"status"`
	Reason     string           `json:"reason,omitempty"`
}

func (l *Ledger) WithdrawFreeLiquidity(vaultID VaultID, accountID AccountID, amount Amount) (*Withdrawal, error) {
	vault, err := l.Vault(vaultID)
	if err != nil {
		return nil, err
	}
	account, err := l.Account(accountID)
	if err != nil {
		return nil, err
	}
	id := l.NextWithdrawalID(vaultID, accountID)
	freeBefore := vault.FreeLiquidity()
	withdrawal := &Withdrawal{
		ID:         id,
		Vault:      vaultID,
		Account:    accountID,
		Asset:      vault.Asset,
		Amount:     amount,
		FreeBefore: freeBefore,
		Clock:      l.Clock,
		Status:     WithdrawalQueued,
	}
	if l.ActiveCycle != nil && l.ActiveCycle.Status == CycleClosed && l.Clock < l.ActiveCycle.ClosedClock+1 {
		withdrawal.Status = WithdrawalRejected
		withdrawal.Reason = ErrWithdrawalWindowClosed.Error()
		l.Withdrawals[id] = withdrawal
		return withdrawal, ErrWithdrawalWindowClosed
	}
	if err := vault.Withdraw(amount); err != nil {
		withdrawal.Status = WithdrawalRejected
		withdrawal.Reason = err.Error()
		withdrawal.FreeAfter = vault.FreeLiquidity()
		l.Withdrawals[id] = withdrawal
		l.Events.Add(l.Clock, "withdrawal.rejected", id.String(), vault.Asset, amount, map[string]interface{}{
			"vault":  vaultID.String(),
			"reason": err.Error(),
		})
		return withdrawal, err
	}
	if err := account.Deposit(vault.Asset, amount); err != nil {
		withdrawal.Status = WithdrawalRejected
		withdrawal.Reason = err.Error()
		l.Withdrawals[id] = withdrawal
		return withdrawal, err
	}
	withdrawal.Status = WithdrawalCompleted
	withdrawal.FreeAfter = vault.FreeLiquidity()
	l.Withdrawals[id] = withdrawal
	l.Events.Add(l.Clock, "withdrawal.completed", id.String(), vault.Asset, amount, map[string]interface{}{
		"vault":   vaultID.String(),
		"account": accountID.String(),
	})
	return withdrawal, nil
}
