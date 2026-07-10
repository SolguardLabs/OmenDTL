package main

import "errors"

var (
	ErrAmountOverflow         = errors.New("amount overflow")
	ErrAmountUnderflow        = errors.New("amount underflow")
	ErrInvalidAmount          = errors.New("invalid amount")
	ErrUnknownAsset           = errors.New("unknown asset")
	ErrUnknownAccount         = errors.New("unknown account")
	ErrUnknownVault           = errors.New("unknown vault")
	ErrUnknownRoute           = errors.New("unknown route")
	ErrUnknownForecast        = errors.New("unknown forecast")
	ErrUnknownReservation     = errors.New("unknown reservation")
	ErrUnknownSettlement      = errors.New("unknown settlement")
	ErrInvalidRouteAsset      = errors.New("invalid route asset")
	ErrInvalidVaultAsset      = errors.New("invalid vault asset")
	ErrInsufficientLiquidity  = errors.New("insufficient liquidity")
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrCycleAlreadyOpen       = errors.New("cycle already open")
	ErrNoOpenCycle            = errors.New("no open cycle")
	ErrCycleClosed            = errors.New("cycle closed")
	ErrForecastAlreadyExists  = errors.New("forecast already exists")
	ErrReservationClosed      = errors.New("reservation closed")
	ErrSettlementNotReady     = errors.New("settlement not ready")
	ErrDemandUpdateRejected   = errors.New("demand update rejected")
	ErrInvalidScenario        = errors.New("invalid scenario")
	ErrInvariantViolation     = errors.New("invariant violation")
	ErrWithdrawalWindowClosed = errors.New("withdrawal window closed")
	ErrForecastWindowMismatch = errors.New("forecast window mismatch")
)

type OmenError struct {
	Op  string `json:"op"`
	Err error  `json:"-"`
	Msg string `json:"message"`
}

func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &OmenError{Op: op, Err: err, Msg: err.Error()}
}

func (e *OmenError) Error() string {
	if e == nil {
		return ""
	}
	if e.Op == "" {
		return e.Msg
	}
	return e.Op + ": " + e.Msg
}

func (e *OmenError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
