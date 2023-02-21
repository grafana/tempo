// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

import (
	"strings"
)

// redisTruncationMark is used as suffix by tracing libraries to indicate that a
// command was truncated.
const redisTruncationMark = "..."

const maxRedisNbCommands = 3

// Redis commands consisting in 2 words
var redisCompoundCommandSet = map[string]bool{
	"CLIENT": true, "CLUSTER": true, "COMMAND": true, "CONFIG": true, "DEBUG": true, "SCRIPT": true}

// QuantizeRedisString returns a quantized version of a Redis query.
//
// TODO(gbbr): Refactor this method to use the tokenizer and
// remove "compactWhitespaces". This method is buggy when commands
// contain quoted strings with newlines.
func (*Obfuscator) QuantizeRedisString(query string) string {
	query = compactWhitespaces(query)

	var resource strings.Builder
	truncated := false
	nbCmds := 0

	for len(query) > 0 && nbCmds < maxRedisNbCommands {
		var rawLine string

		// Read the next command
		idx := strings.IndexByte(query, '\n')
		if idx == -1 {
			rawLine = query
			query = ""
		} else {
			rawLine = query[:idx]
			query = query[idx+1:]
		}

		line := strings.Trim(rawLine, " ")
		if len(line) == 0 {
			continue
		}

		// Parse arguments
		args := strings.SplitN(line, " ", 3)

		if strings.HasSuffix(args[0], redisTruncationMark) {
			truncated = true
			continue
		}

		command := strings.ToUpper(args[0])

		if redisCompoundCommandSet[command] && len(args) > 1 {
			if strings.HasSuffix(args[1], redisTruncationMark) {
				truncated = true
				continue
			}

			command += " " + strings.ToUpper(args[1])
		}

		// Write the command representation
		resource.WriteByte(' ')
		resource.WriteString(command)

		nbCmds++
		truncated = false
	}

	if nbCmds == maxRedisNbCommands || truncated {
		resource.WriteString(" ...")
	}

	return strings.Trim(resource.String(), " ")
}

// ObfuscateRedisString obfuscates the given Redis command.
func (*Obfuscator) ObfuscateRedisString(rediscmd string) string {
	t := newRedisTokenizer([]byte(rediscmd))
	var (
		str  strings.Builder
		cmd  string
		args []string
	)
	for {
		tok, typ, done := t.scan()
		switch typ {
		case redisTokenCommand:
			// new command starting
			if cmd != "" {
				// a previous command was buffered, obfuscate it
				obfuscateRedisCmd(&str, cmd, args...)
				str.WriteByte('\n')
			}
			cmd = tok
			args = args[:0]
		case redisTokenArgument:
			args = append(args, tok)
		}
		if done {
			// last command
			obfuscateRedisCmd(&str, cmd, args...)
			break
		}
	}
	return str.String()
}

func obfuscateRedisCmd(out *strings.Builder, cmd string, args ...string) {
	out.WriteString(cmd)
	if len(args) == 0 {
		return
	}
	out.WriteByte(' ')

	switch strings.ToUpper(cmd) {
	case "AUTH":
		// Obfuscate everything after command
		// • AUTH password
		if len(args) > 0 {
			args[0] = "?"
			args = args[:1]
		}

	case "APPEND", "GETSET", "LPUSHX", "GEORADIUSBYMEMBER", "RPUSHX",
		"SET", "SETNX", "SISMEMBER", "ZRANK", "ZREVRANK", "ZSCORE":
		// Obfuscate 2nd argument:
		// • APPEND key value
		// • GETSET key value
		// • LPUSHX key value
		// • GEORADIUSBYMEMBER key member radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count] [ASC|DESC] [STORE key] [STOREDIST key]
		// • RPUSHX key value
		// • SET key value [expiration EX seconds|PX milliseconds] [NX|XX]
		// • SETNX key value
		// • SISMEMBER key member
		// • ZRANK key member
		// • ZREVRANK key member
		// • ZSCORE key member
		obfuscateRedisArgN(args, 1)

	case "HSET", "HSETNX", "LREM", "LSET", "SETBIT", "SETEX", "PSETEX",
		"SETRANGE", "ZINCRBY", "SMOVE", "RESTORE":
		// Obfuscate 3rd argument:
		// • HSET key field value
		// • HSETNX key field value
		// • LREM key count value
		// • LSET key index value
		// • SETBIT key offset value
		// • SETEX key seconds value
		// • PSETEX key milliseconds value
		// • SETRANGE key offset value
		// • ZINCRBY key increment member
		// • SMOVE source destination member
		// • RESTORE key ttl serialized-value [REPLACE]
		obfuscateRedisArgN(args, 2)

	case "LINSERT":
		// Obfuscate 4th argument:
		// • LINSERT key BEFORE|AFTER pivot value
		obfuscateRedisArgN(args, 3)

	case "GEOHASH", "GEOPOS", "GEODIST", "LPUSH", "RPUSH", "SREM",
		"ZREM", "SADD":
		// Obfuscate all arguments after the first one.
		// • GEOHASH key member [member ...]
		// • GEOPOS key member [member ...]
		// • GEODIST key member1 member2 [unit]
		// • LPUSH key value [value ...]
		// • RPUSH key value [value ...]
		// • SREM key member [member ...]
		// • ZREM key member [member ...]
		// • SADD key member [member ...]
		if len(args) > 1 {
			args[1] = "?"
			args = args[:2]
		}

	case "GEOADD":
		// Obfuscating every 3rd argument starting from first
		// • GEOADD key longitude latitude member [longitude latitude member ...]
		obfuscateRedisArgsStep(args, 1, 3)

	case "HMSET":
		// Every 2nd argument starting from first.
		// • HMSET key field value [field value ...]
		obfuscateRedisArgsStep(args, 1, 2)

	case "MSET", "MSETNX":
		// Every 2nd argument starting from command.
		// • MSET key value [key value ...]
		// • MSETNX key value [key value ...]
		obfuscateRedisArgsStep(args, 0, 2)

	case "CONFIG":
		// Obfuscate 2nd argument to SET sub-command.
		// • CONFIG SET parameter value
		if strings.ToUpper(args[0]) == "SET" {
			obfuscateRedisArgN(args, 2)
		}

	case "BITFIELD":
		// Obfuscate 3rd argument to SET sub-command:
		// • BITFIELD key [GET type offset] [SET type offset value] [INCRBY type offset increment] [OVERFLOW WRAP|SAT|FAIL]
		var n int
		for i, arg := range args {
			if strings.ToUpper(arg) == "SET" {
				n = i
			}
			if n > 0 && i-n == 3 {
				args[i] = "?"
				break
			}
		}

	case "ZADD":
		// Obfuscate every 2nd argument after potential optional ones.
		// • ZADD key [NX|XX] [CH] [INCR] score member [score member ...]
		var i int
	loop:
		for i = range args {
			if i == 0 {
				continue // key
			}
			switch args[i] {
			case "NX", "XX", "CH", "INCR":
				// continue
			default:
				break loop
			}
		}
		obfuscateRedisArgsStep(args, i, 2)

	default:
		// Obfuscate nothing.
	}
	out.WriteString(strings.Join(args, " "))
}

func obfuscateRedisArgN(args []string, n int) {
	if len(args) > n {
		args[n] = "?"
	}
}

func obfuscateRedisArgsStep(args []string, start, step int) {
	if start+step-1 >= len(args) {
		// can't reach target
		return
	}
	for i := start + step - 1; i < len(args); i += step {
		args[i] = "?"
	}
}
