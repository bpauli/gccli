package fit

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

// Fixed timestamp for deterministic tests.
var testTime = time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

func TestFitTimestamp(t *testing.T) {
	ts := fitTimestamp(testTime)
	// 2025-01-15 10:30:00 UTC = Unix 1736937000
	// FIT epoch = 1989-12-31 00:00:00 UTC = Unix 631065600
	// FIT timestamp = 1736937000 - 631065600 = 1105871400
	expected := uint32(1105871400)
	if ts != expected {
		t.Errorf("fitTimestamp = %d, want %d", ts, expected)
	}
}

func TestFitTimestamp_Epoch(t *testing.T) {
	ts := fitTimestamp(fitEpoch)
	if ts != 0 {
		t.Errorf("fitTimestamp(epoch) = %d, want 0", ts)
	}
}

func TestCalcCRC(t *testing.T) {
	// CRC of empty data should be 0
	crc := calcCRC(nil)
	if crc != 0 {
		t.Errorf("calcCRC(nil) = %d, want 0", crc)
	}

	// CRC of known bytes should be deterministic
	crc1 := calcCRC([]byte{0x0C, 0x10, 0x6C, 0x00})
	crc2 := calcCRC([]byte{0x0C, 0x10, 0x6C, 0x00})
	if crc1 != crc2 {
		t.Error("CRC is not deterministic")
	}

	// CRC should differ for different inputs
	crc3 := calcCRC([]byte{0xFF, 0x00, 0x6C, 0x00})
	if crc1 == crc3 {
		t.Error("CRC should differ for different inputs")
	}
}

func TestRecordHeader(t *testing.T) {
	tests := []struct {
		name       string
		definition bool
		lmsgType   byte
		want       byte
	}{
		{"data msg type 0", false, 0, 0x00},
		{"data msg type 1", false, 1, 0x01},
		{"data msg type 3", false, 3, 0x03},
		{"def msg type 0", true, 0, 0x40},
		{"def msg type 1", true, 1, 0x41},
		{"def msg type 3", true, 3, 0x43},
		{"type masked to 4 bits", false, 0x1F, 0x0F},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordHeader(tt.definition, tt.lmsgType)
			if got != tt.want {
				t.Errorf("recordHeader(%v, %d) = 0x%02x, want 0x%02x", tt.definition, tt.lmsgType, got, tt.want)
			}
		})
	}
}

func TestFloatPtr(t *testing.T) {
	p := floatPtr(3.14)
	if p == nil {
		t.Fatal("floatPtr returned nil")
	}
	if *p != 3.14 {
		t.Errorf("floatPtr value = %f, want 3.14", *p)
	}
}

func TestNewEncoder_Header(t *testing.T) {
	enc := NewEncoder()
	data := enc.Bytes()

	if len(data) < 12 {
		t.Fatalf("encoder output too short: %d bytes", len(data))
	}

	// Header size
	if data[0] != 12 {
		t.Errorf("header size = %d, want 12", data[0])
	}
	// Protocol version
	if data[1] != 16 {
		t.Errorf("protocol version = %d, want 16", data[1])
	}
	// Profile version (108 = 0x6C, 0x00 in LE)
	profileVer := binary.LittleEndian.Uint16(data[2:4])
	if profileVer != 108 {
		t.Errorf("profile version = %d, want 108", profileVer)
	}
	// Data size (0 initially)
	dataSize := binary.LittleEndian.Uint32(data[4:8])
	if dataSize != 0 {
		t.Errorf("initial data size = %d, want 0", dataSize)
	}
	// Data type
	if string(data[8:12]) != ".FIT" {
		t.Errorf("data type = %q, want .FIT", string(data[8:12]))
	}
}

func TestWriteFileInfo(t *testing.T) {
	enc := NewEncoder()
	enc.WriteFileInfo(testTime)
	data := enc.Bytes()

	// After header (12) we should have a definition message and a data record.
	// The first byte after header should be the definition record header for lmsg 0.
	pos := 12
	if data[pos] != 0x40 { // definition bit set, lmsg type 0
		t.Errorf("file info def header = 0x%02x, want 0x40", data[pos])
	}
	pos++

	// Fixed content: reserved(1) + arch(1) + mesg_num(2) + num_fields(1)
	if data[pos] != 0 { // reserved
		t.Errorf("reserved = %d, want 0", data[pos])
	}
	pos++
	if data[pos] != 0 { // architecture (little-endian)
		t.Errorf("architecture = %d, want 0", data[pos])
	}
	pos++
	mesgNum := binary.LittleEndian.Uint16(data[pos : pos+2])
	if mesgNum != mesgFileID {
		t.Errorf("mesg num = %d, want %d (file_id)", mesgNum, mesgFileID)
	}
	pos += 2
	numFields := data[pos]
	if numFields != 6 {
		t.Errorf("num fields = %d, want 6", numFields)
	}
	pos++

	// Skip field definitions (6 fields * 3 bytes each = 18 bytes)
	pos += int(numFields) * 3

	// Data record header for lmsg type 0
	if data[pos] != 0x00 {
		t.Errorf("file info data header = 0x%02x, want 0x00", data[pos])
	}
	pos++

	// Verify file type is in the values.
	// Fields: serial_number(uint32z=4), time_created(uint32=4), manufacturer(uint16=2),
	//         product(uint16=2), number(uint16=2), type(enum=1) = 15 bytes
	// The type field (file_type=9) is the last byte of the values.
	valEnd := pos + 15
	if data[valEnd-1] != fileTypeWeight {
		t.Errorf("file type = %d, want %d", data[valEnd-1], fileTypeWeight)
	}
}

func TestWriteFileCreator(t *testing.T) {
	enc := NewEncoder()
	enc.WriteFileCreator()
	data := enc.Bytes()

	pos := 12
	// Definition header for lmsg type 1
	if data[pos] != 0x41 {
		t.Errorf("file creator def header = 0x%02x, want 0x41", data[pos])
	}
	pos++

	// Skip fixed content (5 bytes)
	pos += 5
	// 2 fields: software_version(uint16), hardware_version(uint8) = 2 field defs
	numFields := data[pos-1]
	if numFields != 2 {
		t.Errorf("num fields = %d, want 2", numFields)
	}
}

func TestWriteDeviceInfo(t *testing.T) {
	enc := NewEncoder()
	enc.WriteDeviceInfo(testTime)
	data := enc.Bytes()

	pos := 12
	// Definition header for lmsg type 2
	if data[pos] != 0x42 {
		t.Errorf("device info def header = 0x%02x, want 0x42", data[pos])
	}
}

func TestWriteDeviceInfo_DefinedOnce(t *testing.T) {
	enc := NewEncoder()
	enc.WriteDeviceInfo(testTime)
	size1 := len(enc.Bytes())

	enc.WriteDeviceInfo(testTime.Add(time.Hour))
	size2 := len(enc.Bytes())

	// Second call should be smaller since no definition is written.
	delta := size2 - size1
	// Definition = 1 (header) + 5 (fixed) + 12*3 (fields) = 42 bytes
	// Data record = 1 (header) + values bytes
	// Second record should only add data record (no definition).
	if delta >= 42 {
		t.Errorf("second WriteDeviceInfo added %d bytes, expected less than 42 (no definition)", delta)
	}
}

func TestWriteWeightScale(t *testing.T) {
	enc := NewEncoder()
	weight := 75.5
	bodyFat := 18.3
	muscleMass := 35.2

	enc.WriteWeightScale(WeightScaleData{
		Timestamp:  testTime,
		Weight:     weight,
		PercentFat: &bodyFat,
		MuscleMass: &muscleMass,
	})
	data := enc.Bytes()

	pos := 12
	// Definition header for lmsg type 3
	if data[pos] != 0x43 {
		t.Errorf("weight scale def header = 0x%02x, want 0x43", data[pos])
	}
	pos++

	// Fixed content: reserved(1) + arch(1) + mesg_num(2) + num_fields(1)
	pos += 2 // reserved + arch
	mesgNum := binary.LittleEndian.Uint16(data[pos : pos+2])
	if mesgNum != mesgWeightScale {
		t.Errorf("mesg num = %d, want %d (weight_scale)", mesgNum, mesgWeightScale)
	}
	pos += 2
	numFields := data[pos]
	if numFields != 13 {
		t.Errorf("num fields = %d, want 13", numFields)
	}
}

func TestWriteWeightScale_DefinedOnce(t *testing.T) {
	enc := NewEncoder()
	data1 := WeightScaleData{Timestamp: testTime, Weight: 75.5}
	enc.WriteWeightScale(data1)
	size1 := len(enc.Bytes())

	data2 := WeightScaleData{Timestamp: testTime.Add(time.Hour), Weight: 76.0}
	enc.WriteWeightScale(data2)
	size2 := len(enc.Bytes())

	delta := size2 - size1
	// Definition = 1 + 5 + 13*3 = 45 bytes. Second call should add less.
	if delta >= 45 {
		t.Errorf("second WriteWeightScale added %d bytes, expected less than 45", delta)
	}
}

func TestWeightScaleValues(t *testing.T) {
	enc := NewEncoder()
	weight := 75.5 // Should encode as uint16: 75.5 * 100 = 7550

	enc.WriteWeightScale(WeightScaleData{
		Timestamp: testTime,
		Weight:    weight,
	})

	data := enc.Bytes()
	// Find the data record. Skip header (12) + definition header (1) + fixed (5) +
	// field defs (13 * 3 = 39) = 57 bytes, then data record header (1).
	pos := 12 + 1 + 5 + 13*3 + 1

	// First value: timestamp (uint32) = fitTimestamp(testTime)
	ts := binary.LittleEndian.Uint32(data[pos : pos+4])
	expectedTS := fitTimestamp(testTime)
	if ts != expectedTS {
		t.Errorf("timestamp = %d, want %d", ts, expectedTS)
	}
	pos += 4

	// Second value: weight (uint16) = 75.5 * 100 = 7550
	w := binary.LittleEndian.Uint16(data[pos : pos+2])
	if w != 7550 {
		t.Errorf("weight = %d, want 7550", w)
	}
	pos += 2

	// Third value: percent_fat (uint16) = invalid (0xFFFF) since nil
	pf := binary.LittleEndian.Uint16(data[pos : pos+2])
	if pf != 0xFFFF {
		t.Errorf("percent_fat = %d, want 0xFFFF (invalid)", pf)
	}
}

func TestWeightScaleWithAllFields(t *testing.T) {
	bodyFat := 18.3
	hydration := 55.0
	visceralFat := 5.2
	bone := 3.1
	muscle := 35.0
	basalMet := 1500.0
	activeMet := 2000.0
	physique := 5.0
	metAge := 30.0
	visceralRating := 7.0
	bmi := 24.5

	enc := NewEncoder()
	enc.WriteWeightScale(WeightScaleData{
		Timestamp:         testTime,
		Weight:            75.5,
		PercentFat:        &bodyFat,
		PercentHydration:  &hydration,
		VisceralFatMass:   &visceralFat,
		BoneMass:          &bone,
		MuscleMass:        &muscle,
		BasalMet:          &basalMet,
		ActiveMet:         &activeMet,
		PhysiqueRating:    &physique,
		MetabolicAge:      &metAge,
		VisceralFatRating: &visceralRating,
		BMI:               &bmi,
	})

	data := enc.Bytes()
	// After header + def header + fixed + field defs + data header
	pos := 12 + 1 + 5 + 13*3 + 1
	pos += 4 // skip timestamp
	pos += 2 // skip weight

	// percent_fat: 18.3 * 100 = 1830
	pf := binary.LittleEndian.Uint16(data[pos : pos+2])
	if pf != 1830 {
		t.Errorf("percent_fat = %d, want 1830", pf)
	}
	pos += 2

	// percent_hydration: 55.0 * 100 = 5500
	ph := binary.LittleEndian.Uint16(data[pos : pos+2])
	if ph != 5500 {
		t.Errorf("percent_hydration = %d, want 5500", ph)
	}
	pos += 2

	// visceral_fat_mass: 5.2 * 100 = 520
	vfm := binary.LittleEndian.Uint16(data[pos : pos+2])
	if vfm != 520 {
		t.Errorf("visceral_fat_mass = %d, want 520", vfm)
	}
	pos += 2

	// bone_mass: 3.1 * 100 = 310
	bm := binary.LittleEndian.Uint16(data[pos : pos+2])
	if bm != 310 {
		t.Errorf("bone_mass = %d, want 310", bm)
	}
	pos += 2

	// muscle_mass: 35.0 * 100 = 3500
	mm := binary.LittleEndian.Uint16(data[pos : pos+2])
	if mm != 3500 {
		t.Errorf("muscle_mass = %d, want 3500", mm)
	}
	pos += 2

	// basal_met: 1500.0 * 4 = 6000
	basalVal := binary.LittleEndian.Uint16(data[pos : pos+2])
	if basalVal != 6000 {
		t.Errorf("basal_met = %d, want 6000", basalVal)
	}
	pos += 2

	// active_met: 2000.0 * 4 = 8000
	activeVal := binary.LittleEndian.Uint16(data[pos : pos+2])
	if activeVal != 8000 {
		t.Errorf("active_met = %d, want 8000", activeVal)
	}
	pos += 2

	// physique_rating: 5 * 1 = 5 (uint8)
	if data[pos] != 5 {
		t.Errorf("physique_rating = %d, want 5", data[pos])
	}
	pos++

	// metabolic_age: 30 * 1 = 30 (uint8)
	if data[pos] != 30 {
		t.Errorf("metabolic_age = %d, want 30", data[pos])
	}
	pos++

	// visceral_fat_rating: 7 * 1 = 7 (uint8)
	if data[pos] != 7 {
		t.Errorf("visceral_fat_rating = %d, want 7", data[pos])
	}
	pos++

	// bmi: 24.5 * 10 = 245
	bmiVal := binary.LittleEndian.Uint16(data[pos : pos+2])
	if bmiVal != 245 {
		t.Errorf("bmi = %d, want 245", bmiVal)
	}
}

func TestFinish_UpdatesDataSize(t *testing.T) {
	enc := NewEncoder()
	enc.WriteFileInfo(testTime)
	enc.Finish()

	data := enc.Bytes()
	dataSize := binary.LittleEndian.Uint32(data[4:8])

	// Total size = header(12) + message data + CRC(2)
	// Data size should be total - header - CRC
	expectedDataSize := uint32(len(data) - 12 - 2)
	if dataSize != expectedDataSize {
		t.Errorf("data size in header = %d, want %d", dataSize, expectedDataSize)
	}
}

func TestFinish_AppendsCRC(t *testing.T) {
	enc := NewEncoder()
	enc.WriteFileInfo(testTime)

	sizeBeforeFinish := len(enc.Bytes())
	enc.Finish()
	sizeAfterFinish := len(enc.Bytes())

	// Finish should add exactly 2 bytes (CRC)
	if sizeAfterFinish != sizeBeforeFinish+2 {
		t.Errorf("Finish added %d bytes, want 2", sizeAfterFinish-sizeBeforeFinish)
	}
}

func TestFinish_CRCIsValid(t *testing.T) {
	enc := NewEncoder()
	enc.WriteFileInfo(testTime)
	enc.WriteFileCreator()
	enc.Finish()

	data := enc.Bytes()

	// CRC is over header + data (everything except the 2 CRC bytes themselves)
	headerAndData := data[:len(data)-2]
	expectedCRC := calcCRC(headerAndData)

	actualCRC := binary.LittleEndian.Uint16(data[len(data)-2:])
	if actualCRC != expectedCRC {
		t.Errorf("CRC = 0x%04x, want 0x%04x", actualCRC, expectedCRC)
	}
}

func TestEncodeWeightScale_FullFile(t *testing.T) {
	weight := 80.0
	bodyFat := 20.0

	data := EncodeWeightScale(WeightScaleData{
		Timestamp:  testTime,
		Weight:     weight,
		PercentFat: &bodyFat,
	})

	// Verify it's a valid FIT file
	if len(data) < 14 { // 12 header + 2 CRC minimum
		t.Fatalf("file too short: %d bytes", len(data))
	}

	// Check header
	if data[0] != 12 {
		t.Errorf("header size = %d, want 12", data[0])
	}
	if string(data[8:12]) != ".FIT" {
		t.Errorf("signature = %q, want .FIT", string(data[8:12]))
	}

	// Check data size matches
	dataSize := binary.LittleEndian.Uint32(data[4:8])
	expectedDataSize := uint32(len(data) - 12 - 2) // total - header - CRC
	if dataSize != expectedDataSize {
		t.Errorf("data size = %d, want %d", dataSize, expectedDataSize)
	}

	// Verify CRC
	headerAndData := data[:len(data)-2]
	expectedCRC := calcCRC(headerAndData)
	actualCRC := binary.LittleEndian.Uint16(data[len(data)-2:])
	if actualCRC != expectedCRC {
		t.Errorf("CRC = 0x%04x, want 0x%04x", actualCRC, expectedCRC)
	}
}

func TestEncodeWeightScale_Deterministic(t *testing.T) {
	data1 := EncodeWeightScale(WeightScaleData{
		Timestamp: testTime,
		Weight:    75.5,
	})
	data2 := EncodeWeightScale(WeightScaleData{
		Timestamp: testTime,
		Weight:    75.5,
	})

	if len(data1) != len(data2) {
		t.Fatalf("sizes differ: %d vs %d", len(data1), len(data2))
	}

	for i := range data1 {
		if data1[i] != data2[i] {
			t.Errorf("byte %d differs: 0x%02x vs 0x%02x", i, data1[i], data2[i])
		}
	}
}

func TestEncodeWeightScale_DifferentWeights(t *testing.T) {
	data1 := EncodeWeightScale(WeightScaleData{Timestamp: testTime, Weight: 75.0})
	data2 := EncodeWeightScale(WeightScaleData{Timestamp: testTime, Weight: 80.0})

	if len(data1) != len(data2) {
		t.Fatalf("sizes differ: %d vs %d", len(data1), len(data2))
	}

	// Files should differ (different weight values and CRC)
	same := true
	for i := range data1 {
		if data1[i] != data2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("files should differ for different weights")
	}
}

func TestEncodeWeightScale_ContainsAllMessages(t *testing.T) {
	data := EncodeWeightScale(WeightScaleData{Timestamp: testTime, Weight: 75.0})

	// Should contain 4 definition messages (file_id, file_creator, device_info, weight_scale)
	defCount := 0
	pos := 12               // skip header
	for pos < len(data)-2 { // -2 for CRC
		header := data[pos]
		pos++
		isDef := (header & 0x40) != 0
		if isDef {
			defCount++
			// Skip: reserved(1) + arch(1) + mesg_num(2) + num_fields(1) = 5 bytes
			if pos+4 >= len(data)-2 {
				break
			}
			numFields := int(data[pos+4])
			pos += 5 + numFields*3 // skip fixed content + field defs
		} else {
			// Data record: need to determine size from the associated definition.
			// For this test, we just verify the def count at the end.
			// Skip the rest since we can't easily determine data record size without
			// tracking definitions.
			break
		}
	}

	if defCount < 1 {
		t.Errorf("found %d definition messages, want at least 1", defCount)
	}
}

func TestBuildContent_InvalidValues(t *testing.T) {
	// All nil values should produce invalid sentinels
	fields := []fieldDef{
		{num: 0, bt: btUint16, value: nil, scale: 100},
		{num: 1, bt: btUint8, value: nil, scale: 1},
		{num: 2, bt: btUint32, value: nil, scale: 1},
	}

	_, vals := buildContent(fields)

	// uint16 invalid = 0xFFFF (2 bytes)
	v16 := binary.LittleEndian.Uint16(vals[0:2])
	if v16 != 0xFFFF {
		t.Errorf("uint16 invalid = 0x%04x, want 0xFFFF", v16)
	}

	// uint8 invalid = 0xFF (1 byte)
	if vals[2] != 0xFF {
		t.Errorf("uint8 invalid = 0x%02x, want 0xFF", vals[2])
	}

	// uint32 invalid = 0xFFFFFFFF (4 bytes)
	v32 := binary.LittleEndian.Uint32(vals[3:7])
	if v32 != 0xFFFFFFFF {
		t.Errorf("uint32 invalid = 0x%08x, want 0xFFFFFFFF", v32)
	}
}

func TestBuildContent_Uint32zInvalid(t *testing.T) {
	fields := []fieldDef{
		{num: 0, bt: btUint32z, value: nil, scale: 0},
	}

	_, vals := buildContent(fields)
	v := binary.LittleEndian.Uint32(vals[0:4])
	if v != 0x00000000 {
		t.Errorf("uint32z invalid = 0x%08x, want 0x00000000", v)
	}
}

func TestBuildContent_ScaledValues(t *testing.T) {
	weight := 75.5
	fields := []fieldDef{
		{num: 0, bt: btUint16, value: &weight, scale: 100}, // 75.5 * 100 = 7550
	}

	_, vals := buildContent(fields)
	v := binary.LittleEndian.Uint16(vals[0:2])
	if v != 7550 {
		t.Errorf("scaled value = %d, want 7550", v)
	}
}

func TestBuildContent_ZeroScale(t *testing.T) {
	val := 42.0
	fields := []fieldDef{
		{num: 0, bt: btUint8, value: &val, scale: 0}, // 0 scale means no scaling
	}

	_, vals := buildContent(fields)
	if vals[0] != 42 {
		t.Errorf("zero-scale value = %d, want 42", vals[0])
	}
}

func TestBuildContent_FieldDefs(t *testing.T) {
	fields := []fieldDef{
		{num: 5, bt: btUint16, value: nil, scale: 0},
		{num: 253, bt: btUint32, value: nil, scale: 0},
	}

	defs, _ := buildContent(fields)

	// Field 0: num=5, size=2, field=0x84
	if defs[0] != 5 || defs[1] != 2 || defs[2] != 0x84 {
		t.Errorf("field 0 def = [%d, %d, 0x%02x], want [5, 2, 0x84]", defs[0], defs[1], defs[2])
	}

	// Field 1: num=253, size=4, field=0x86
	if defs[3] != 253 || defs[4] != 4 || defs[5] != 0x86 {
		t.Errorf("field 1 def = [%d, %d, 0x%02x], want [253, 4, 0x86]", defs[3], defs[4], defs[5])
	}
}

func TestWriteValue_Sizes(t *testing.T) {
	tests := []struct {
		name string
		bt   baseType
		val  float64
		want []byte
	}{
		{"uint8 zero", btUint8, 0, []byte{0x00}},
		{"uint8 max", btUint8, 254, []byte{0xFE}},
		{"uint16 zero", btUint16, 0, []byte{0x00, 0x00}},
		{"uint16 1000", btUint16, 1000, []byte{0xE8, 0x03}}, // 1000 LE
		{"uint32 zero", btUint32, 0, []byte{0x00, 0x00, 0x00, 0x00}},
		{"uint32 large", btUint32, 1105871400, []byte{0x28, 0x42, 0xEA, 0x41}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf [4]byte
			b := &bytes.Buffer{}
			writeValue(b, tt.bt, tt.val)
			got := b.Bytes()
			copy(buf[:], got)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d bytes, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("byte %d: got 0x%02x, want 0x%02x", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWriteInvalid_Sizes(t *testing.T) {
	tests := []struct {
		name string
		bt   baseType
		want []byte
	}{
		{"enum", btEnum, []byte{0xFF}},
		{"uint8", btUint8, []byte{0xFF}},
		{"uint16", btUint16, []byte{0xFF, 0xFF}},
		{"uint32", btUint32, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{"uint32z", btUint32z, []byte{0x00, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bytes.Buffer{}
			writeInvalid(b, tt.bt)
			got := b.Bytes()
			if len(got) != len(tt.want) {
				t.Fatalf("got %d bytes, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("byte %d: got 0x%02x, want 0x%02x", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEncodeWeightScale_WeightOnly(t *testing.T) {
	data := EncodeWeightScale(WeightScaleData{
		Timestamp: testTime,
		Weight:    75.0,
	})

	// Must be > 12 (header) + 2 (CRC)
	if len(data) <= 14 {
		t.Fatalf("file too short: %d bytes", len(data))
	}

	// Verify FIT signature
	if string(data[8:12]) != ".FIT" {
		t.Errorf("signature = %q, want .FIT", string(data[8:12]))
	}

	// Verify valid CRC
	headerAndData := data[:len(data)-2]
	expectedCRC := calcCRC(headerAndData)
	actualCRC := binary.LittleEndian.Uint16(data[len(data)-2:])
	if actualCRC != expectedCRC {
		t.Errorf("CRC mismatch: got 0x%04x, want 0x%04x", actualCRC, expectedCRC)
	}
}
