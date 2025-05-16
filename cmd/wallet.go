package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
)

func Bech32ToEthAddress(bech32Addr string) (string, error) {

	hrp, data, err := bech32.Decode(bech32Addr)
	if err != nil {
		return "", fmt.Errorf("failed to decode bech32 address: %w", err)
	}
	fmt.Println("HRP:", hrp)

	// Convert from 5-bit groups back to 8-bit bytes
	decoded, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %w", err)
	}
	if len(decoded) != 20 {
		return "", fmt.Errorf("expected 20 bytes, got %d", len(decoded))
	}

	ethAddress := "0x" + strings.ToLower(hex.EncodeToString(decoded))
	return ethAddress, nil
}
