// SPDX-FileCopyrightText: 2024 Shun Sakai
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package lzip

import "errors"

// ErrInvalidMagic represents an error due to the magic number was invalid.
var ErrInvalidMagic = errors.New("lzip: invalid magic number")

// UnsupportedVersionError represents an error due to the version number stored
// in the header indicated the lzip format which is not supported by this
// package.
type UnsupportedVersionError struct {
	// Version represents the obtained version number.
	Version byte
}

// Error returns a string representation of an [UnsupportedVersionError].
func (e *UnsupportedVersionError) Error() string {
	return "lzip: unsupported version number"
}

// UnknownVersionError represents an error due to the version number stored in
// the header was not recognized by this package.
type UnknownVersionError struct {
	// Version represents the obtained version number.
	Version byte
}

// Error returns a string representation of an [UnknownVersionError].
func (e *UnknownVersionError) Error() string {
	return "lzip: unknown version number"
}

// DictSizeTooSmallError represents an error due to the dictionary size was
// smaller than 4 KiB.
type DictSizeTooSmallError struct {
	// DictSize represents the obtained dictionary size.
	DictSize uint32
}

// Error returns a string representation of a [DictSizeTooSmallError].
func (e *DictSizeTooSmallError) Error() string {
	return "lzip: dictionary size is too small"
}

// DictSizeTooLargeError represents an error due to the dictionary size was
// larger than 512 MiB.
type DictSizeTooLargeError struct {
	// DictSize represents the obtained dictionary size.
	DictSize uint32
}

// Error returns a string representation of a [DictSizeTooLargeError].
func (e *DictSizeTooLargeError) Error() string {
	return "lzip: dictionary size is too large"
}

// InvalidCRCError represents an error due to a CRC of the original
// uncompressed data mismatched.
type InvalidCRCError struct {
	// CRC represents the obtained CRC.
	CRC uint32
}

// Error returns a string representation of an [InvalidCRCError].
func (e *InvalidCRCError) Error() string {
	return "lzip: CRC mismatch"
}

// InvalidDataSizeError represents an error due to the size of the original
// uncompressed data stored in the trailer and the actual size of it mismatched.
type InvalidDataSizeError struct {
	// DataSize represents the obtained data size.
	DataSize uint64
}

// Error returns a string representation of an [InvalidDataSizeError].
func (e *InvalidDataSizeError) Error() string {
	return "lzip: data size mismatch"
}

// InvalidMemberSizeError represents an error due to the total size of the
// member stored in the trailer and the actual total size of it mismatched.
type InvalidMemberSizeError struct {
	// MemberSize represents the obtained member size.
	MemberSize uint64
}

// Error returns a string representation of an [InvalidMemberSizeError].
func (e *InvalidMemberSizeError) Error() string {
	return "lzip: member size mismatch"
}

// MemberSizeTooLargeError represents an error due to the member size was
// larger than 2 PiB.
type MemberSizeTooLargeError struct {
	// MemberSize represents the obtained member size.
	MemberSize uint64
}

// Error returns a string representation of a [MemberSizeTooLargeError].
func (e *MemberSizeTooLargeError) Error() string {
	return "lzip: member size is too large"
}
