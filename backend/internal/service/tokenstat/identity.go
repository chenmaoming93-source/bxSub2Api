package tokenstat

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
)

const CanonicalEncodingVersion byte = 1

type DimensionIdentity struct {
	Canonical []byte
	Hash      [16]byte
}

func BuildDimensionIdentity(codes []DimensionCode, values map[DimensionCode]DimensionValue) (DimensionIdentity, error) {
	canonicalCodes, err := CanonicalDimensionCodes(codes)
	if err != nil {
		return DimensionIdentity{}, err
	}
	var buffer bytes.Buffer
	buffer.WriteByte(CanonicalEncodingVersion)
	for _, code := range canonicalCodes {
		definition, _ := Dimension(code)
		value, ok := values[code]
		if !ok {
			return DimensionIdentity{}, fmt.Errorf("missing dimension %q", code)
		}
		if err := validateDimensionValue(definition, value); err != nil {
			return DimensionIdentity{}, err
		}
		writePart(&buffer, []byte(code))
		buffer.WriteByte(byte(definition.Version))
		buffer.WriteByte(valueTypeTag(value.Type))
		switch value.Type {
		case ValueTypeInt64:
			writePart(&buffer, []byte(strconv.FormatInt(value.Int64, 10)))
		case ValueTypeString:
			writePart(&buffer, []byte(value.String))
		}
	}
	sum := sha256.Sum256(buffer.Bytes())
	var hash [16]byte
	copy(hash[:], sum[:16])
	return DimensionIdentity{Canonical: buffer.Bytes(), Hash: hash}, nil
}

func (i DimensionIdentity) HashHex() string {
	return hex.EncodeToString(i.Hash[:])
}

func writePart(buffer *bytes.Buffer, value []byte) {
	_ = binary.Write(buffer, binary.BigEndian, uint32(len(value)))
	_, _ = buffer.Write(value)
}

func valueTypeTag(valueType ValueType) byte {
	if valueType == ValueTypeInt64 {
		return 1
	}
	return 2
}
