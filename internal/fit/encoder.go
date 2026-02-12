package fit

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"
)

// FIT epoch: UTC 00:00 Dec 31 1989 (631065600 Unix seconds).
var fitEpoch = time.Date(1989, 12, 31, 0, 0, 0, 0, time.UTC)

// crcTable is the CRC-16 lookup table for FIT protocol.
var crcTable = [16]uint16{
	0x0000, 0xCC01, 0xD801, 0x1400,
	0xF001, 0x3C00, 0x2800, 0xE401,
	0xA001, 0x6C00, 0x7800, 0xB401,
	0x5000, 0x9C01, 0x8801, 0x4400,
}

// Global message numbers.
const (
	mesgFileID      uint16 = 0
	mesgDeviceInfo  uint16 = 23
	mesgWeightScale uint16 = 30
	mesgFileCreator uint16 = 49
)

// Local message types.
const (
	lmsgFileInfo    byte = 0
	lmsgFileCreator byte = 1
	lmsgDeviceInfo  byte = 2
	lmsgWeightScale byte = 3
)

// File type for weight.
const fileTypeWeight byte = 9

// baseType represents a FIT base type with its encoding properties.
type baseType struct {
	field   byte
	size    byte
	invalid uint64
}

// FIT base types.
var (
	btEnum    = baseType{field: 0x00, size: 1, invalid: 0xFF}
	btUint8   = baseType{field: 0x02, size: 1, invalid: 0xFF}
	btUint16  = baseType{field: 0x84, size: 2, invalid: 0xFFFF}
	btUint32  = baseType{field: 0x86, size: 4, invalid: 0xFFFFFFFF}
	btUint32z = baseType{field: 0x8C, size: 4, invalid: 0x00000000}
)

// fieldDef describes a single field in a FIT message definition.
type fieldDef struct {
	num   byte
	bt    baseType
	value *float64
	scale float64
}

// WeightScaleData holds the data for a weight scale FIT record.
type WeightScaleData struct {
	Timestamp         time.Time
	Weight            float64  // kg
	PercentFat        *float64 // percent
	PercentHydration  *float64 // percent
	VisceralFatMass   *float64 // kg
	BoneMass          *float64 // kg
	MuscleMass        *float64 // kg
	BasalMet          *float64 // kcal/day
	ActiveMet         *float64 // kcal/day
	PhysiqueRating    *float64
	MetabolicAge      *float64
	VisceralFatRating *float64
	BMI               *float64
}

// Encoder writes body composition data into the FIT binary format.
type Encoder struct {
	buf                bytes.Buffer
	deviceInfoDefined  bool
	weightScaleDefined bool
}

// NewEncoder creates a new FIT Encoder and writes the initial header.
func NewEncoder() *Encoder {
	e := &Encoder{}
	e.writeHeader(0)
	return e
}

// writeHeader writes the 12-byte FIT file header.
func (e *Encoder) writeHeader(dataSize uint32) {
	e.buf.Reset()
	var header [12]byte
	header[0] = 12                                  // header size
	header[1] = 16                                  // protocol version (1.0)
	binary.LittleEndian.PutUint16(header[2:4], 108) // profile version
	binary.LittleEndian.PutUint32(header[4:8], dataSize)
	copy(header[8:12], ".FIT")
	_, _ = e.buf.Write(header[:])
}

// WriteFileInfo writes the file_id message (definition + data record).
func (e *Encoder) WriteFileInfo(t time.Time) {
	ts := fitTimestamp(t)

	fields := []fieldDef{
		{num: 3, bt: btUint32z, value: nil, scale: 0},                            // serial_number
		{num: 4, bt: btUint32, value: floatPtr(float64(ts)), scale: 0},           // time_created
		{num: 1, bt: btUint16, value: nil, scale: 0},                             // manufacturer
		{num: 2, bt: btUint16, value: nil, scale: 0},                             // product
		{num: 5, bt: btUint16, value: nil, scale: 0},                             // number
		{num: 0, bt: btEnum, value: floatPtr(float64(fileTypeWeight)), scale: 0}, // type
	}

	fieldDefs, values := buildContent(fields)

	// Definition message
	e.buf.WriteByte(recordHeader(true, lmsgFileInfo))
	writeFixedContent(&e.buf, mesgFileID, len(fields))
	_, _ = e.buf.Write(fieldDefs)

	// Data record
	e.buf.WriteByte(recordHeader(false, lmsgFileInfo))
	_, _ = e.buf.Write(values)
}

// WriteFileCreator writes the file_creator message (definition + data record).
func (e *Encoder) WriteFileCreator() {
	fields := []fieldDef{
		{num: 0, bt: btUint16, value: nil, scale: 0}, // software_version
		{num: 1, bt: btUint8, value: nil, scale: 0},  // hardware_version
	}

	fieldDefs, values := buildContent(fields)

	// Definition message
	e.buf.WriteByte(recordHeader(true, lmsgFileCreator))
	writeFixedContent(&e.buf, mesgFileCreator, len(fields))
	_, _ = e.buf.Write(fieldDefs)

	// Data record
	e.buf.WriteByte(recordHeader(false, lmsgFileCreator))
	_, _ = e.buf.Write(values)
}

// WriteDeviceInfo writes a device_info message. The definition is written only once.
func (e *Encoder) WriteDeviceInfo(t time.Time) {
	ts := fitTimestamp(t)

	fields := []fieldDef{
		{num: 253, bt: btUint32, value: floatPtr(float64(ts)), scale: 1}, // timestamp
		{num: 3, bt: btUint32z, value: nil, scale: 1},                    // serial_number
		{num: 7, bt: btUint32, value: nil, scale: 1},                     // cum_operating_time
		{num: 8, bt: btUint32, value: nil, scale: 0},                     // unknown field
		{num: 2, bt: btUint16, value: nil, scale: 1},                     // manufacturer
		{num: 4, bt: btUint16, value: nil, scale: 1},                     // product
		{num: 5, bt: btUint16, value: nil, scale: 100},                   // software_version
		{num: 10, bt: btUint16, value: nil, scale: 256},                  // battery_voltage
		{num: 0, bt: btUint8, value: nil, scale: 1},                      // device_index
		{num: 1, bt: btUint8, value: nil, scale: 1},                      // device_type
		{num: 6, bt: btUint8, value: nil, scale: 1},                      // hardware_version
		{num: 11, bt: btUint8, value: nil, scale: 0},                     // battery_status
	}

	fieldDefs, values := buildContent(fields)

	if !e.deviceInfoDefined {
		e.buf.WriteByte(recordHeader(true, lmsgDeviceInfo))
		writeFixedContent(&e.buf, mesgDeviceInfo, len(fields))
		_, _ = e.buf.Write(fieldDefs)
		e.deviceInfoDefined = true
	}

	e.buf.WriteByte(recordHeader(false, lmsgDeviceInfo))
	_, _ = e.buf.Write(values)
}

// WriteWeightScale writes a weight_scale message. The definition is written only once.
func (e *Encoder) WriteWeightScale(data WeightScaleData) {
	ts := fitTimestamp(data.Timestamp)
	weight := floatPtr(data.Weight)

	fields := []fieldDef{
		{num: 253, bt: btUint32, value: floatPtr(float64(ts)), scale: 1}, // timestamp
		{num: 0, bt: btUint16, value: weight, scale: 100},                // weight
		{num: 1, bt: btUint16, value: data.PercentFat, scale: 100},       // percent_fat
		{num: 2, bt: btUint16, value: data.PercentHydration, scale: 100}, // percent_hydration
		{num: 3, bt: btUint16, value: data.VisceralFatMass, scale: 100},  // visceral_fat_mass
		{num: 4, bt: btUint16, value: data.BoneMass, scale: 100},         // bone_mass
		{num: 5, bt: btUint16, value: data.MuscleMass, scale: 100},       // muscle_mass
		{num: 7, bt: btUint16, value: data.BasalMet, scale: 4},           // basal_met
		{num: 9, bt: btUint16, value: data.ActiveMet, scale: 4},          // active_met
		{num: 8, bt: btUint8, value: data.PhysiqueRating, scale: 1},      // physique_rating
		{num: 10, bt: btUint8, value: data.MetabolicAge, scale: 1},       // metabolic_age
		{num: 11, bt: btUint8, value: data.VisceralFatRating, scale: 1},  // visceral_fat_rating
		{num: 13, bt: btUint16, value: data.BMI, scale: 10},              // bmi
	}

	fieldDefs, values := buildContent(fields)

	if !e.weightScaleDefined {
		e.buf.WriteByte(recordHeader(true, lmsgWeightScale))
		writeFixedContent(&e.buf, mesgWeightScale, len(fields))
		_, _ = e.buf.Write(fieldDefs)
		e.weightScaleDefined = true
	}

	e.buf.WriteByte(recordHeader(false, lmsgWeightScale))
	_, _ = e.buf.Write(values)
}

// Finish rewrites the header with the correct data size, then appends the CRC.
func (e *Encoder) Finish() {
	totalSize := e.buf.Len()
	dataSize := uint32(totalSize - 12) // subtract header

	// Rewrite header with correct data size
	data := e.buf.Bytes()
	binary.LittleEndian.PutUint32(data[4:8], dataSize)

	// Calculate CRC over all bytes (header + data)
	crc := calcCRC(data)
	_ = binary.Write(&e.buf, binary.LittleEndian, crc)
}

// Bytes returns the complete FIT file bytes.
func (e *Encoder) Bytes() []byte {
	return e.buf.Bytes()
}

// EncodeWeightScale is a convenience function that encodes a complete weight scale
// FIT file from the given data.
func EncodeWeightScale(data WeightScaleData) []byte {
	enc := NewEncoder()
	enc.WriteFileInfo(data.Timestamp)
	enc.WriteFileCreator()
	enc.WriteDeviceInfo(data.Timestamp)
	enc.WriteWeightScale(data)
	enc.Finish()
	return enc.Bytes()
}

// --- helpers ---

// fitTimestamp converts a Go time to a FIT timestamp (seconds since FIT epoch).
func fitTimestamp(t time.Time) uint32 {
	return uint32(t.Unix() - fitEpoch.Unix())
}

// floatPtr returns a pointer to a float64.
func floatPtr(v float64) *float64 {
	return &v
}

// recordHeader builds a 1-byte record header.
// bit 6 set = definition message, bits 0-3 = local message type.
func recordHeader(definition bool, lmsgType byte) byte {
	var h byte
	if definition {
		h = 1 << 6
	}
	return h | (lmsgType & 0x0F)
}

// writeFixedContent writes the fixed portion of a definition message:
// reserved(1) + architecture(1) + global message number(2) + num fields(1).
// Architecture 0 = little-endian.
func writeFixedContent(buf *bytes.Buffer, mesgNum uint16, numFields int) {
	buf.WriteByte(0) // reserved
	buf.WriteByte(0) // architecture: little-endian
	var mn [2]byte
	binary.LittleEndian.PutUint16(mn[:], mesgNum)
	_, _ = buf.Write(mn[:])
	buf.WriteByte(byte(numFields))
}

// buildContent builds field definitions and encoded values from a list of fieldDefs.
func buildContent(fields []fieldDef) (defs []byte, vals []byte) {
	var defBuf, valBuf bytes.Buffer

	for _, f := range fields {
		// Field definition: field_num(1) + size(1) + base_type(1)
		defBuf.WriteByte(f.num)
		defBuf.WriteByte(f.bt.size)
		defBuf.WriteByte(f.bt.field)

		// Value encoding
		if f.value == nil {
			writeInvalid(&valBuf, f.bt)
		} else {
			v := *f.value
			if f.scale > 0 {
				v *= f.scale
			}
			writeValue(&valBuf, f.bt, v)
		}
	}

	return defBuf.Bytes(), valBuf.Bytes()
}

// writeValue encodes a numeric value according to its base type.
func writeValue(buf *bytes.Buffer, bt baseType, v float64) {
	switch bt.size {
	case 1:
		buf.WriteByte(byte(math.Round(v)))
	case 2:
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(math.Round(v)))
		_, _ = buf.Write(b[:])
	case 4:
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(math.Round(v)))
		_, _ = buf.Write(b[:])
	}
}

// writeInvalid writes the invalid sentinel value for a base type.
func writeInvalid(buf *bytes.Buffer, bt baseType) {
	switch bt.size {
	case 1:
		buf.WriteByte(byte(bt.invalid))
	case 2:
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(bt.invalid))
		_, _ = buf.Write(b[:])
	case 4:
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(bt.invalid))
		_, _ = buf.Write(b[:])
	}
}

// calcCRC computes the FIT CRC-16 over all bytes.
func calcCRC(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		// lower nibble
		tmp := crcTable[crc&0x0F]
		crc = (crc >> 4) & 0x0FFF
		crc = crc ^ tmp ^ crcTable[b&0x0F]
		// upper nibble
		tmp = crcTable[crc&0x0F]
		crc = (crc >> 4) & 0x0FFF
		crc = crc ^ tmp ^ crcTable[(b>>4)&0x0F]
	}
	return crc
}
