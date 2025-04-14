package tokenaccess

import (
	"errors"
	"std"

	"gno.land/p/demo/grc/grc20"
)

var (
	ErrInsufficientBalance   = errors.New("access denied: insufficient token balance")
	ErrInsufficientAllowance = errors.New("access denied: insufficient allowance")
)

// RequireMinimumBalance ensures a caller has at least amount of tokens.
func RequireMinimumBalance(token *grc20.Token, amount uint64, caller std.Address) error {
	balance := token.BalanceOf(caller)
	if balance < amount {
		return ErrInsufficientBalance
	}
	return nil
}

// RequireMinimumAllowance ensures a caller has an approved allowance from the owner.
func RequireMinimumAllowance(
	token *grc20.Token,
	owner std.Address,
	amount uint64,
	caller std.Address,
) error {
	allowance := token.Allowance(owner, caller)
	if allowance < amount {
		return ErrInsufficientAllowance
	}
	return nil
}

// RewardAndApprove mints rewardAmt tokens to owner and increases target's allowance by allowanceIncrement.
func RewardAndApprove(
	owner std.Address,
	rewardAmt uint64,
	token *grc20.Token,
	ledger *grc20.PrivateLedger,
	target std.Address,
	allowanceIncrement uint64,
) {
	currentAllowance := token.Allowance(owner, target)
	newAllowance := currentAllowance + allowanceIncrement

	if err := ledger.Mint(owner, rewardAmt); err != nil {
		panic("token mint failed: " + err.Error())
	}

	if err := ledger.Approve(owner, target, newAllowance); err != nil {
		panic("token approval failed: " + err.Error())
	}
}

// Spend transfers amount tokens from from to to, respecting allowance.
func Spend(
	token *grc20.Token,
	ledger *grc20.PrivateLedger,
	from std.Address,
	to std.Address,
	amount uint64,
) error {
	err := ledger.TransferFrom(from, to, to, amount)
	if err != nil {
		return errors.New("token transfer failed: " + err.Error())
	}
	return nil
}

// HasAccess returns true if the caller meets both balance and allowance requirements.
func HasAccess(
	token *grc20.Token,
	owner std.Address,
	caller std.Address,
	minBalance uint64,
	minAllowance uint64,
) bool {
	return token.BalanceOf(caller) >= minBalance &&
		token.Allowance(owner, caller) >= minAllowance
}
