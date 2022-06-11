//go:build purego || !amd64

package parquet

func optimize(int) bool { return false }
