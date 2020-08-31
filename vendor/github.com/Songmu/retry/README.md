retry
=======

[![Build Status](https://travis-ci.org/Songmu/retry.png?branch=master)][travis]
[![Coverage Status](https://coveralls.io/repos/Songmu/retry/badge.png?branch=master)][coveralls]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]
[![GoDoc](https://godoc.org/github.com/Songmu/retry?status.svg)](godoc)

[travis]: https://travis-ci.org/Songmu/retry
[coveralls]: https://coveralls.io/r/Songmu/retry?branch=master
[license]: https://github.com/Songmu/retry/blob/master/LICENSE
[godoc]: https://godoc.org/github.com/Songmu/retry

## Description

retry N times

It is golang porting of perl's [Sub::Retry](https://metacpan.org/release/Sub-Retry)

## Synopsis

    err := retry.Retry(3, 1*time.Second, func() error {
        // return error once in a while
    })
    if err != nil {
        // error handling
    }

## Author

[Songmu](https://github.com/Songmu)
