package gofakeit

import (
	"math"
	"math/rand"
)

// Number will generate a random number between given min And max
func Number(min int, max int) int { return number(globalFaker.Rand, min, max) }

// Number will generate a random number between given min And max
func (f *Faker) Number(min int, max int) int { return number(f.Rand, min, max) }

func number(r *rand.Rand, min int, max int) int { return randIntRange(r, min, max) }

// Uint8 will generate a random uint8 value
func Uint8() uint8 { return uint8Func(globalFaker.Rand) }

// Uint8 will generate a random uint8 value
func (f *Faker) Uint8() uint8 { return uint8Func(f.Rand) }

func uint8Func(r *rand.Rand) uint8 { return uint8(randUintRange(r, minUint, math.MaxUint8)) }

// Uint16 will generate a random uint16 value
func Uint16() uint16 { return uint16Func(globalFaker.Rand) }

// Uint16 will generate a random uint16 value
func (f *Faker) Uint16() uint16 { return uint16Func(f.Rand) }

func uint16Func(r *rand.Rand) uint16 { return uint16(randUintRange(r, minUint, math.MaxUint16)) }

// Uint32 will generate a random uint32 value
func Uint32() uint32 { return uint32Func(globalFaker.Rand) }

// Uint32 will generate a random uint32 value
func (f *Faker) Uint32() uint32 { return uint32Func(f.Rand) }

func uint32Func(r *rand.Rand) uint32 { return uint32(randUintRange(r, minUint, math.MaxUint32)) }

// Uint64 will generate a random uint64 value
func Uint64() uint64 { return uint64Func(globalFaker.Rand) }

// Uint64 will generate a random uint64 value
func (f *Faker) Uint64() uint64 { return uint64Func(f.Rand) }

func uint64Func(r *rand.Rand) uint64 { return uint64(randUintRange(r, minUint, maxUint)) }

// UintRange will generate a random uint value between min and max
func UintRange(min, max uint) uint { return uintRangeFunc(globalFaker.Rand, min, max) }

// UintRange will generate a random uint value between min and max
func (f *Faker) UintRange(min, max uint) uint { return uintRangeFunc(f.Rand, min, max) }

func uintRangeFunc(r *rand.Rand, min, max uint) uint { return randUintRange(r, min, max) }

// Int8 will generate a random Int8 value
func Int8() int8 { return int8Func(globalFaker.Rand) }

// Int8 will generate a random Int8 value
func (f *Faker) Int8() int8 { return int8Func(f.Rand) }

func int8Func(r *rand.Rand) int8 { return int8(randIntRange(r, math.MinInt8, math.MaxInt8)) }

// Int16 will generate a random int16 value
func Int16() int16 { return int16Func(globalFaker.Rand) }

// Int16 will generate a random int16 value
func (f *Faker) Int16() int16 { return int16Func(f.Rand) }

func int16Func(r *rand.Rand) int16 { return int16(randIntRange(r, math.MinInt16, math.MaxInt16)) }

// Int32 will generate a random int32 value
func Int32() int32 { return int32Func(globalFaker.Rand) }

// Int32 will generate a random int32 value
func (f *Faker) Int32() int32 { return int32Func(f.Rand) }

func int32Func(r *rand.Rand) int32 { return int32(randIntRange(r, math.MinInt32, math.MaxInt32)) }

// Int64 will generate a random int64 value
func Int64() int64 { return int64Func(globalFaker.Rand) }

// Int64 will generate a random int64 value
func (f *Faker) Int64() int64 { return int64Func(f.Rand) }

func int64Func(r *rand.Rand) int64 { return int64(randIntRange(r, minInt, maxInt)) }

// IntRange will generate a random int value between min and max
func IntRange(min, max int) int { return intRangeFunc(globalFaker.Rand, min, max) }

// IntRange will generate a random int value between min and max
func (f *Faker) IntRange(min, max int) int { return intRangeFunc(f.Rand, min, max) }

func intRangeFunc(r *rand.Rand, min, max int) int { return randIntRange(r, min, max) }

// Float32 will generate a random float32 value
func Float32() float32 { return float32Func(globalFaker.Rand) }

// Float32 will generate a random float32 value
func (f *Faker) Float32() float32 { return float32Func(f.Rand) }

func float32Func(r *rand.Rand) float32 {
	return float32Range(r, math.SmallestNonzeroFloat32, math.MaxFloat32)
}

// Float32Range will generate a random float32 value between min and max
func Float32Range(min, max float32) float32 {
	return float32Range(globalFaker.Rand, min, max)
}

// Float32Range will generate a random float32 value between min and max
func (f *Faker) Float32Range(min, max float32) float32 {
	return float32Range(f.Rand, min, max)
}

func float32Range(r *rand.Rand, min, max float32) float32 {
	if min == max {
		return min
	}
	return r.Float32()*(max-min) + min
}

// Float64 will generate a random float64 value
func Float64() float64 {
	return float64Func(globalFaker.Rand)
}

// Float64 will generate a random float64 value
func (f *Faker) Float64() float64 {
	return float64Func(f.Rand)
}

func float64Func(r *rand.Rand) float64 {
	return float64Range(r, math.SmallestNonzeroFloat64, math.MaxFloat64)
}

// Float64Range will generate a random float64 value between min and max
func Float64Range(min, max float64) float64 {
	return float64Range(globalFaker.Rand, min, max)
}

// Float64Range will generate a random float64 value between min and max
func (f *Faker) Float64Range(min, max float64) float64 {
	return float64Range(f.Rand, min, max)
}

func float64Range(r *rand.Rand, min, max float64) float64 {
	if min == max {
		return min
	}
	return r.Float64()*(max-min) + min
}

// ShuffleInts will randomize a slice of ints
func ShuffleInts(a []int) { shuffleInts(globalFaker.Rand, a) }

// ShuffleInts will randomize a slice of ints
func (f *Faker) ShuffleInts(a []int) { shuffleInts(f.Rand, a) }

func shuffleInts(r *rand.Rand, a []int) {
	for i := range a {
		j := r.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

// RandomInt will take in a slice of int and return a randomly selected value
func RandomInt(i []int) int { return randomInt(globalFaker.Rand, i) }

// RandomInt will take in a slice of int and return a randomly selected value
func (f *Faker) RandomInt(i []int) int { return randomInt(f.Rand, i) }

func randomInt(r *rand.Rand, i []int) int {
	size := len(i)
	if size == 0 {
		return 0
	}
	if size == 1 {
		return i[0]
	}
	return i[r.Intn(size)]
}

// RandomUint will take in a slice of uint and return a randomly selected value
func RandomUint(u []uint) uint { return randomUint(globalFaker.Rand, u) }

// RandomUint will take in a slice of uint and return a randomly selected value
func (f *Faker) RandomUint(u []uint) uint { return randomUint(f.Rand, u) }

func randomUint(r *rand.Rand, u []uint) uint {
	size := len(u)
	if size == 0 {
		return 0
	}
	if size == 1 {
		return u[0]
	}
	return u[r.Intn(size)]
}

// HexUint8 will generate a random uint8 hex value with "0x" prefix
func HexUint8() string { return hexUint(globalFaker.Rand, 8) }

// HexUint8 will generate a random uint8 hex value with "0x" prefix
func (f *Faker) HexUint8() string { return hexUint(f.Rand, 8) }

// HexUint16 will generate a random uint16 hex value with "0x" prefix
func HexUint16() string { return hexUint(globalFaker.Rand, 16) }

// HexUint16 will generate a random uint16 hex value with "0x" prefix
func (f *Faker) HexUint16() string { return hexUint(f.Rand, 16) }

// HexUint32 will generate a random uint32 hex value with "0x" prefix
func HexUint32() string { return hexUint(globalFaker.Rand, 32) }

// HexUint32 will generate a random uint32 hex value with "0x" prefix
func (f *Faker) HexUint32() string { return hexUint(f.Rand, 32) }

// HexUint64 will generate a random uint64 hex value with "0x" prefix
func HexUint64() string { return hexUint(globalFaker.Rand, 64) }

// HexUint64 will generate a random uint64 hex value with "0x" prefix
func (f *Faker) HexUint64() string { return hexUint(f.Rand, 64) }

// HexUint128 will generate a random uint128 hex value with "0x" prefix
func HexUint128() string { return hexUint(globalFaker.Rand, 128) }

// HexUint128 will generate a random uint128 hex value with "0x" prefix
func (f *Faker) HexUint128() string { return hexUint(f.Rand, 128) }

// HexUint256 will generate a random uint256 hex value with "0x" prefix
func HexUint256() string { return hexUint(globalFaker.Rand, 256) }

// HexUint256 will generate a random uint256 hex value with "0x" prefix
func (f *Faker) HexUint256() string { return hexUint(f.Rand, 256) }

func hexUint(r *rand.Rand, bitSize int) string {
	digits := []byte("0123456789abcdef")
	hexLen := (bitSize >> 2) + 2
	if hexLen <= 2 {
		return "0x"
	}

	s := make([]byte, hexLen)
	s[0], s[1] = '0', 'x'
	for i := 2; i < hexLen; i++ {
		s[i] = digits[r.Intn(16)]
	}
	return string(s)
}

func addNumberLookup() {
	AddFuncLookup("number", Info{
		Display:     "Number",
		Category:    "number",
		Description: "Mathematical concept used for counting, measuring, and expressing quantities or values",
		Example:     "14866",
		Output:      "int",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "int", Default: "-2147483648", Description: "Minimum integer value"},
			{Field: "max", Display: "Max", Type: "int", Default: "2147483647", Description: "Maximum integer value"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetInt(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetInt(m, "max")
			if err != nil {
				return nil, err
			}

			return number(r, min, max), nil
		},
	})

	AddFuncLookup("uint8", Info{
		Display:     "Uint8",
		Category:    "number",
		Description: "Unsigned 8-bit integer, capable of representing values from 0 to 255",
		Example:     "152",
		Output:      "uint8",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return uint8Func(r), nil
		},
	})

	AddFuncLookup("uint16", Info{
		Display:     "Uint16",
		Category:    "number",
		Description: "Unsigned 16-bit integer, capable of representing values from 0 to 65,535",
		Example:     "34968",
		Output:      "uint16",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return uint16Func(r), nil
		},
	})

	AddFuncLookup("uint32", Info{
		Display:     "Uint32",
		Category:    "number",
		Description: "Unsigned 32-bit integer, capable of representing values from 0 to 4,294,967,295",
		Example:     "1075055705",
		Output:      "uint32",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return uint32Func(r), nil
		},
	})

	AddFuncLookup("uint64", Info{
		Display:     "Uint64",
		Category:    "number",
		Description: "Unsigned 64-bit integer, capable of representing values from 0 to 18,446,744,073,709,551,615",
		Example:     "843730692693298265",
		Output:      "uint64",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return uint64Func(r), nil
		},
	})

	AddFuncLookup("uintrange", Info{
		Display:     "UintRange",
		Category:    "number",
		Description: "Non-negative integer value between given range",
		Example:     "1075055705",
		Output:      "uint",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "uint", Default: "0", Description: "Minimum uint value"},
			{Field: "max", Display: "Max", Type: "uint", Default: "4294967295", Description: "Maximum uint value"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetUint(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetUint(m, "max")
			if err != nil {
				return nil, err
			}

			return uintRangeFunc(r, min, max), nil
		},
	})

	AddFuncLookup("int8", Info{
		Display:     "Int8",
		Category:    "number",
		Description: "Signed 8-bit integer, capable of representing values from -128 to 127",
		Example:     "24",
		Output:      "int8",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return int8Func(r), nil
		},
	})

	AddFuncLookup("int16", Info{
		Display:     "Int16",
		Category:    "number",
		Description: "Signed 16-bit integer, capable of representing values from 32,768 to 32,767",
		Example:     "2200",
		Output:      "int16",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return int16Func(r), nil
		},
	})

	AddFuncLookup("int32", Info{
		Display:     "Int32",
		Category:    "number",
		Description: "Signed 32-bit integer, capable of representing values from -2,147,483,648 to 2,147,483,647",
		Example:     "-1072427943",
		Output:      "int32",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return int32Func(r), nil
		},
	})

	AddFuncLookup("int64", Info{
		Display:     "Int64",
		Category:    "number",
		Description: "Signed 64-bit integer, capable of representing values from -9,223,372,036,854,775,808 to -9,223,372,036,854,775,807",
		Example:     "-8379641344161477543",
		Output:      "int64",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return int64Func(r), nil
		},
	})

	AddFuncLookup("intrange", Info{
		Display:     "IntRange",
		Category:    "number",
		Description: "Integer value between given range",
		Example:     "-8379477543",
		Output:      "int",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "int", Description: "Minimum int value"},
			{Field: "max", Display: "Max", Type: "int", Description: "Maximum int value"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetInt(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetInt(m, "max")
			if err != nil {
				return nil, err
			}

			return intRangeFunc(r, min, max), nil
		},
	})

	AddFuncLookup("float32", Info{
		Display:     "Float32",
		Category:    "number",
		Description: "Data type representing floating-point numbers with 32 bits of precision in computing",
		Example:     "3.1128167e+37",
		Output:      "float32",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return float32Func(r), nil
		},
	})

	AddFuncLookup("float32range", Info{
		Display:     "Float32 Range",
		Category:    "number",
		Description: "Float32 value between given range",
		Example:     "914774.6",
		Output:      "float32",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "float", Description: "Minimum float32 value"},
			{Field: "max", Display: "Max", Type: "float", Description: "Maximum float32 value"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetFloat32(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetFloat32(m, "max")
			if err != nil {
				return nil, err
			}

			return float32Range(r, min, max), nil
		},
	})

	AddFuncLookup("float64", Info{
		Display:     "Float64",
		Category:    "number",
		Description: "Data type representing floating-point numbers with 64 bits of precision in computing",
		Example:     "1.644484108270445e+307",
		Output:      "float64",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return float64Func(r), nil
		},
	})

	AddFuncLookup("float64range", Info{
		Display:     "Float64 Range",
		Category:    "number",
		Description: "Float64 value between given range",
		Example:     "914774.5585333086",
		Output:      "float64",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "float", Description: "Minimum float64 value"},
			{Field: "max", Display: "Max", Type: "float", Description: "Maximum float64 value"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetFloat64(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetFloat64(m, "max")
			if err != nil {
				return nil, err
			}

			return float64Range(r, min, max), nil
		},
	})

	AddFuncLookup("shuffleints", Info{
		Display:     "Shuffle Ints",
		Category:    "number",
		Description: "Shuffles an array of ints",
		Example:     "1,2,3,4 => 3,1,4,2",
		Output:      "[]int",
		Params: []Param{
			{Field: "ints", Display: "Integers", Type: "[]int", Description: "Delimited separated integers"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			ints, err := info.GetIntArray(m, "ints")
			if err != nil {
				return nil, err
			}

			shuffleInts(r, ints)

			return ints, nil
		},
	})

	AddFuncLookup("randomint", Info{
		Display:     "Random Int",
		Category:    "number",
		Description: "Randomly selected value from a slice of int",
		Example:     "-1,2,-3,4 => -3",
		Output:      "int",
		Params: []Param{
			{Field: "ints", Display: "Integers", Type: "[]int", Description: "Delimited separated integers"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			ints, err := info.GetIntArray(m, "ints")
			if err != nil {
				return nil, err
			}

			return randomInt(r, ints), nil
		},
	})

	AddFuncLookup("randomuint", Info{
		Display:     "Random Uint",
		Category:    "number",
		Description: "Randomly selected value from a slice of uint",
		Example:     "1,2,3,4 => 4",
		Output:      "uint",
		Params: []Param{
			{Field: "uints", Display: "Unsigned Integers", Type: "[]uint", Description: "Delimited separated unsigned integers"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			uints, err := info.GetUintArray(m, "uints")
			if err != nil {
				return nil, err
			}

			return randomUint(r, uints), nil
		},
	})

	AddFuncLookup("hexuint8", Info{
		Display:     "HexUint8",
		Category:    "number",
		Description: "Hexadecimal representation of an 8-bit unsigned integer",
		Example:     "0x87",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return hexUint(r, 8), nil
		},
	})

	AddFuncLookup("hexuint16", Info{
		Display:     "HexUint16",
		Category:    "number",
		Description: "Hexadecimal representation of an 16-bit unsigned integer",
		Example:     "0x8754",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return hexUint(r, 16), nil
		},
	})

	AddFuncLookup("hexuint32", Info{
		Display:     "HexUint32",
		Category:    "number",
		Description: "Hexadecimal representation of an 32-bit unsigned integer",
		Example:     "0x87546957",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return hexUint(r, 32), nil
		},
	})

	AddFuncLookup("hexuint64", Info{
		Display:     "HexUint64",
		Category:    "number",
		Description: "Hexadecimal representation of an 64-bit unsigned integer",
		Example:     "0x875469578e51b5e5",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return hexUint(r, 64), nil
		},
	})

	AddFuncLookup("hexuint128", Info{
		Display:     "HexUint128",
		Category:    "number",
		Description: "Hexadecimal representation of an 128-bit unsigned integer",
		Example:     "0x875469578e51b5e56c95b64681d147a1",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return hexUint(r, 128), nil
		},
	})

	AddFuncLookup("hexuint256", Info{
		Display:     "HexUint256",
		Category:    "number",
		Description: "Hexadecimal representation of an 256-bit unsigned integer",
		Example:     "0x875469578e51b5e56c95b64681d147a12cde48a4f417231b0c486abbc263e48d",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return hexUint(r, 256), nil
		},
	})
}
