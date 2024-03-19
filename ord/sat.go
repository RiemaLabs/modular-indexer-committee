package ord

import (
	"fmt"
	"strconv"
	"strings"
)

func (op *OutPoint) Encode() string {
	return fmt.Sprintf("%s:%d", op.txID, op.offset)
}

func DecodeOutPoint(s string) (*OutPoint, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		err := fmt.Errorf("invalid outPoint: %s", s)
		return nil, err
	}
	txID := parts[0]
	if len(txID) != 32 {
		err := fmt.Errorf("invalid txID in outPoint: %s", txID)
		return nil, err
	}
	offset, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	p := OutPoint{
		txID:   TXID(txID),
		offset: offset,
	}
	return &p, err
}

func (sp *SatPoint) Encode() string {
	return fmt.Sprintf("%s:%d", sp.outPoint.Encode(), sp.offset)
}

func DecodeSatPoint(s string) (*SatPoint, error) {
	lastColonIndex := strings.LastIndex(s, ":")
	if lastColonIndex == -1 {
		err := fmt.Errorf("invalid satPoint: %s", s)
		return nil, err
	}
	part1 := s[:lastColonIndex]
	op, err := DecodeOutPoint(part1)
	if err != nil {
		return nil, err
	}
	part2 := s[lastColonIndex+1:]
	offset, err := strconv.ParseUint(part2, 10, 64)
	if err != nil {
		return nil, err
	}
	p := SatPoint{
		outPoint: *op,
		offset:   offset,
	}
	return &p, nil
}