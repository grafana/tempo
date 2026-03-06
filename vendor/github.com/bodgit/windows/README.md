[![GitHub release](https://img.shields.io/github/v/release/bodgit/windows)](https://github.com/bodgit/windows/releases)
[![Build Status](https://img.shields.io/github/workflow/status/bodgit/windows/build)](https://github.com/bodgit/windows/actions?query=workflow%3Abuild)
[![Coverage Status](https://coveralls.io/repos/github/bodgit/windows/badge.svg?branch=main)](https://coveralls.io/github/bodgit/windows?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/bodgit/windows)](https://goreportcard.com/report/github.com/bodgit/windows)
[![GoDoc](https://godoc.org/github.com/bodgit/windows?status.svg)](https://godoc.org/github.com/bodgit/windows)
![Go version](https://img.shields.io/badge/Go-1.18-brightgreen.svg)
![Go version](https://img.shields.io/badge/Go-1.17-brightgreen.svg)

windows
=======

A collection of types native to Windows but are useful on non-Windows platforms.

The `FILETIME`-equivalent type is the sole export which is a 1:1 copy of the type found in the `golang.org/x/sys/windows` package. That package only builds on `GOOS=windows` and this particular type gets used in other protocols and file types such as NTLMv2 and 7-zip.
