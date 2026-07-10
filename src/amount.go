package main

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Amount int64

const (
	ZeroAmount Amount = 0
	BasisPoint Amount = 10_000
)

func NewAmount(value int64) Amount {
	if value < 0 {
		return 0
	}
	return Amount(value)
}

func MustAmount(value int64) Amount {
	if value < 0 {
		panic("negative amount")
	}
	return Amount(value)
}

func (a Amount) Int64() int64 {
	return int64(a)
}

func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
}

func (a Amount) IsZero() bool {
	return a == 0
}

func (a Amount) Positive() bool {
	return a > 0
}

func (a Amount) NonNegative() bool {
	return a >= 0
}

func (a Amount) Add(b Amount) (Amount, error) {
	if b > 0 && a > Amount(^uint64(0)>>1)-b {
		return 0, ErrAmountOverflow
	}
	return a + b, nil
}

func (a Amount) Sub(b Amount) (Amount, error) {
	if b > a {
		return 0, ErrAmountUnderflow
	}
	return a - b, nil
}

func (a Amount) MustAdd(b Amount) Amount {
	next, err := a.Add(b)
	if err != nil {
		panic(err)
	}
	return next
}

func (a Amount) MustSub(b Amount) Amount {
	next, err := a.Sub(b)
	if err != nil {
		panic(err)
	}
	return next
}

func (a Amount) MulBpsFloor(bps Amount) Amount {
	if a <= 0 || bps <= 0 {
		return 0
	}
	return Amount((int64(a) * int64(bps)) / int64(BasisPoint))
}

func (a Amount) MulBpsCeil(bps Amount) Amount {
	if a <= 0 || bps <= 0 {
		return 0
	}
	num := int64(a) * int64(bps)
	den := int64(BasisPoint)
	return Amount((num + den - 1) / den)
}

func (a Amount) MulRatioFloor(numerator Amount, denominator Amount) Amount {
	if a <= 0 || numerator <= 0 || denominator <= 0 {
		return 0
	}
	return Amount((int64(a) * int64(numerator)) / int64(denominator))
}

func (a Amount) MulRatioCeil(numerator Amount, denominator Amount) Amount {
	if a <= 0 || numerator <= 0 || denominator <= 0 {
		return 0
	}
	num := int64(a) * int64(numerator)
	den := int64(denominator)
	return Amount((num + den - 1) / den)
}

func (a Amount) Clamp(min Amount, max Amount) Amount {
	if a < min {
		return min
	}
	if a > max {
		return max
	}
	return a
}

func MinAmount(a Amount, b Amount) Amount {
	if a < b {
		return a
	}
	return b
}

func MaxAmount(a Amount, b Amount) Amount {
	if a > b {
		return a
	}
	return b
}

func SumAmounts(values []Amount) Amount {
	var total Amount
	for _, value := range values {
		total += value
	}
	return total
}

func AbsDelta(a Amount, b Amount) Amount {
	if a >= b {
		return a - b
	}
	return b - a
}

func PercentageOf(part Amount, total Amount) Amount {
	if part <= 0 || total <= 0 {
		return 0
	}
	return Amount((int64(part) * int64(BasisPoint)) / int64(total))
}

func WeightedAmount(amount Amount, weight int) Amount {
	if amount <= 0 || weight <= 0 {
		return 0
	}
	return Amount(int64(amount) * int64(weight))
}

func ParseAmount(value string) (Amount, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidAmount, value)
	}
	if parsed < 0 {
		return 0, ErrInvalidAmount
	}
	return Amount(parsed), nil
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(a), 10)), nil
}

func (a *Amount) UnmarshalJSON(data []byte) error {
	var raw int64
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw < 0 {
		return ErrInvalidAmount
	}
	*a = Amount(raw)
	return nil
}

type AmountBucket struct {
	Asset  AssetID `json:"asset"`
	Amount Amount  `json:"amount"`
}

type AmountBuckets []AmountBucket

func (b AmountBuckets) Clone() AmountBuckets {
	out := make(AmountBuckets, 0, len(b))
	out = append(out, b...)
	return out
}

func (b AmountBuckets) Get(asset AssetID) Amount {
	for _, bucket := range b {
		if bucket.Asset == asset {
			return bucket.Amount
		}
	}
	return 0
}

func (b *AmountBuckets) Add(asset AssetID, amount Amount) {
	if amount <= 0 {
		return
	}
	for i := range *b {
		if (*b)[i].Asset == asset {
			(*b)[i].Amount += amount
			return
		}
	}
	*b = append(*b, AmountBucket{Asset: asset, Amount: amount})
}

func (b *AmountBuckets) Sub(asset AssetID, amount Amount) error {
	if amount <= 0 {
		return nil
	}
	for i := range *b {
		if (*b)[i].Asset == asset {
			if (*b)[i].Amount < amount {
				return ErrAmountUnderflow
			}
			(*b)[i].Amount -= amount
			return nil
		}
	}
	return ErrAmountUnderflow
}

func (b AmountBuckets) Total() Amount {
	var total Amount
	for _, bucket := range b {
		total += bucket.Amount
	}
	return total
}

func (b AmountBuckets) NonNegative() bool {
	for _, bucket := range b {
		if bucket.Amount < 0 {
			return false
		}
	}
	return true
}
