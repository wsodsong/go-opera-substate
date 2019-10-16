package inter

import (
	"math"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/go-lachesis/common/littleendian"
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

const (
	MaxUint24 = uint64(math.MaxUint8) * math.MaxUint16
	MaxUint40 = uint64(math.MaxUint8) * math.MaxUint32
	MaxUint48 = uint64(math.MaxUint16) * math.MaxUint32
	MaxUint56 = MaxUint48 * math.MaxUint8
)

func (e *EventHeaderData) MarshalBinary() ([]byte, error) {
	// Calculate size of constant sized fields
	length := 53 + common.AddressLength + 2*common.HashLength

	// Calculate sizes of slice fields
	parentsCount := 0
	if e.Parents != nil {
		parentsCount = len(e.Parents)
	}

	extraCount := 0
	if e.Extra != nil {
		extraCount = len(e.Extra)
	}

	// Full length with data about sizes of slices
	length += 4 + parentsCount*common.HashLength + 4 + extraCount

	buf := make([]byte, length*2, length*2)

	// Simple types values

	// Detect max value from 4 fields
	b1 := maxBytesForUint32(e.Version)
	b2 := maxBytesForUint32(uint32(e.Epoch))
	b3 := maxBytesForUint32(uint32(e.Seq))
	b4 := maxBytesForUint32(uint32(e.Frame))
	sizeByte := byte((b1 - 1) | ((b2 - 1) << 2) | ((b3 - 1) << 4) | ((b4 - 1) << 6))

	offset := 0

	copy(buf[offset:offset+1], []byte{sizeByte})
	offset++

	copy(buf[offset:offset+int(b1)], littleendian.Int32ToBytes(e.Version))
	offset += int(b1)
	copy(buf[offset:offset+int(b2)], littleendian.Int32ToBytes(uint32(e.Epoch)))
	offset += int(b2)
	copy(buf[offset:offset+int(b3)], littleendian.Int32ToBytes(uint32(e.Seq)))
	offset += int(b3)
	copy(buf[offset:offset+int(b4)], littleendian.Int32ToBytes(uint32(e.Frame)))
	offset += int(b4)

	b1 = maxBytesForUint32(uint32(e.Lamport))
	sizeByte = byte(b1 - 1)
	copy(buf[offset:offset+1], []byte{sizeByte})
	offset++
	copy(buf[offset:offset+int(b1)], littleendian.Int32ToBytes(uint32(e.Lamport)))
	offset += int(b1)

	b1 = maxBytesForUint64(e.GasPowerLeft)
	b2 = maxBytesForUint64(e.GasPowerUsed)
	sizeByte = byte((b1 - 1) | ((b2 - 1) << 4))
	copy(buf[offset:offset+1], []byte{sizeByte})
	offset++
	copy(buf[offset:offset+int(b1)], littleendian.Int64ToBytes(e.GasPowerLeft))
	offset += int(b1)
	copy(buf[offset:offset+int(b2)], littleendian.Int64ToBytes(e.GasPowerUsed))
	offset += int(b2)

	b1 = maxBytesForUint64(uint64(e.ClaimedTime))
	b2 = maxBytesForUint64(uint64(e.MedianTime))
	sizeByte = byte((b1 - 1) | ((b2 - 1) << 4))
	copy(buf[offset:offset+1], []byte{sizeByte})
	offset++
	copy(buf[offset:offset+int(b1)], littleendian.Int64ToBytes(uint64(e.ClaimedTime)))
	offset += int(b1)
	copy(buf[offset:offset+int(b2)], littleendian.Int64ToBytes(uint64(e.MedianTime)))
	offset += int(b2)

	// Fixed types []byte values
	copy(buf[offset:offset+common.AddressLength], e.Creator.Bytes())
	offset += common.AddressLength
	copy(buf[offset:offset+common.HashLength], e.PrevEpochHash.Bytes())
	offset += common.HashLength
	copy(buf[offset:offset+common.HashLength], e.TxHash.Bytes())
	offset += common.HashLength

	// boolean
	b := byte(0)
	if e.IsRoot {
		b = 1
	}
	copy(buf[offset:offset+1], []byte{b})
	offset++

	// Sliced values
	copy(buf[offset:offset+4], littleendian.Int32ToBytes(uint32(parentsCount)))
	offset += 4

	// Save epoch from first Parents (assume - all Parens have equal epoch part)
	if parentsCount > 0 {
		copy(buf[offset:offset+4], e.Parents[0].Bytes())
		offset += 4
	}

	for _, ev := range e.Parents {
		copy(buf[offset:offset+common.HashLength-4], ev.Bytes()[4:common.HashLength])
		offset += common.HashLength - 4
	}

	copy(buf[offset:offset+4], littleendian.Int32ToBytes(uint32(extraCount)))
	offset += 4
	copy(buf[offset:offset+extraCount], e.Extra)
	offset += extraCount

	bufLimited := buf[0:offset]

	return bufLimited, nil
}

func (e *EventHeaderData) UnmarshalBinary(buf []byte) error {
	// Simple types values
	offset := 0
	sizeByte := buf[offset]
	b1 := sizeByte&3 + 1
	b2 := (sizeByte>>2)&3 + 1
	b3 := (sizeByte>>4)&3 + 1
	b4 := (sizeByte>>6)&3 + 1
	offset++

	uint32buf := make([]byte, 4, 4)

	copy(uint32buf, buf[offset:offset+int(b1)])
	e.Version = littleendian.BytesToInt32(uint32buf)
	offset += int(b1)

	copy(uint32buf, []byte{0, 0, 0, 0})
	copy(uint32buf, buf[offset:offset+int(b2)])
	e.Epoch = idx.Epoch(littleendian.BytesToInt32(uint32buf))
	offset += int(b2)

	copy(uint32buf, []byte{0, 0, 0, 0})
	copy(uint32buf, buf[offset:offset+int(b3)])
	e.Seq = idx.Event(littleendian.BytesToInt32(uint32buf))
	offset += int(b3)

	copy(uint32buf, []byte{0, 0, 0, 0})
	copy(uint32buf, buf[offset:offset+int(b4)])
	e.Frame = idx.Frame(littleendian.BytesToInt32(uint32buf))
	offset += int(b4)

	sizeByte = buf[offset]
	b1 = sizeByte&3 + 1
	offset++

	copy(uint32buf, []byte{0, 0, 0, 0})
	copy(uint32buf, buf[offset:offset+int(b1)])
	e.Lamport = idx.Lamport(littleendian.BytesToInt32(uint32buf))
	offset += int(b1)

	sizeByte = buf[offset]
	b1 = sizeByte&7 + 1
	b2 = (sizeByte>>4)&7 + 1
	offset++

	uint64buf := make([]byte, 8, 8)

	copy(uint64buf, buf[offset:offset+int(b1)])
	e.GasPowerLeft = littleendian.BytesToInt64(uint64buf)
	offset += int(b1)

	copy(uint64buf, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	copy(uint64buf, buf[offset:offset+int(b2)])
	e.GasPowerUsed = littleendian.BytesToInt64(uint64buf)
	offset += int(b2)

	sizeByte = buf[offset]
	b1 = sizeByte&7 + 1
	b2 = (sizeByte>>4)&7 + 1
	offset++

	copy(uint64buf, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	copy(uint64buf, buf[offset:offset+int(b1)])
	e.ClaimedTime = Timestamp(littleendian.BytesToInt64(uint64buf))
	offset += int(b1)

	copy(uint64buf, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	copy(uint64buf, buf[offset:offset+int(b2)])
	e.MedianTime = Timestamp(littleendian.BytesToInt64(uint64buf))
	offset += int(b2)

	// Fixed types []byte values
	e.Creator.SetBytes(buf[offset : offset+common.AddressLength])
	offset += common.AddressLength
	e.PrevEpochHash.SetBytes(buf[offset : offset+common.HashLength])
	offset += common.HashLength
	e.TxHash.SetBytes(buf[offset : offset+common.HashLength])
	offset += common.HashLength

	// Boolean
	e.IsRoot = buf[offset] != byte(0)
	offset += 1

	// Sliced values
	parentsCount := littleendian.BytesToInt32(buf[offset : offset+4])

	offset += 4

	evBuf := make([]byte, common.HashLength)
	if parentsCount > 0 {
		// Read epoch for all Parents
		copy(evBuf[0:4], buf[offset:offset+4])
		offset += 4
	}

	e.Parents = make(hash.Events, parentsCount, parentsCount)
	for i := 0; i < int(parentsCount); i++ {
		ev := hash.Event{}

		copy(evBuf[4:common.HashLength], buf[offset:offset+common.HashLength-4])
		ev.SetBytes(evBuf)
		offset += common.HashLength - 4

		e.Parents[i] = ev
	}

	extraCount := littleendian.BytesToInt32(buf[offset : offset+4])
	offset += 4
	e.Extra = make([]byte, extraCount, extraCount)
	copy(e.Extra, buf[offset:offset+int(extraCount)])

	return nil
}

func maxBytesForUint32(t uint32) uint {
	return maxBytesForUint64(uint64(t))
}

func maxBytesForUint64(t uint64) uint {
	if t <= math.MaxUint8 {
		return 1
	}
	if t <= math.MaxUint16 {
		return 2
	}
	if t <= MaxUint24 {
		return 3
	}
	if t <= math.MaxUint32 {
		return 4
	}
	if t <= MaxUint40 {
		return 5
	}
	if t <= MaxUint48 {
		return 6
	}
	if t <= MaxUint56 {
		return 7
	}
	return 8
}
