// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package log

import "strings"

// redirectLogger is used to redirect klog logs to datadog logs. klog is
// client-go's logger, logging to STDERR by default, which makes all severities
// into ERROR, along with the formatting just being off. To make the
// conversion, we set a redirectLogger as klog's output, and parse the severity
// and log message out of every log line.
// NOTE: on klog v2 this parsing is no longer necessary, as it allows us to use
// kSetLogger() instead of kSetOutputBySeverity(). unfortunately we
// still have some dependencies stuck on v1, so we keep the parsing.
type KlogRedirectLogger struct {
	stackDepth int
}

func NewKlogRedirectLogger(stackDepth int) KlogRedirectLogger {
	return KlogRedirectLogger{
		stackDepth: stackDepth,
	}
}

func (l KlogRedirectLogger) Write(b []byte) (int, error) {
	// klog log lines have the following format:
	//     Lmmdd hh:mm:ss.uuuuuu threadid file:line] msg...
	// so we parse L to decide in which level to log, and we try to find
	// the ']' character, to ignore anything up to that point, as we don't
	// care about the header outside of the log level.

	msg := string(b)

	i := strings.IndexByte(msg, ']')
	if i >= 0 {
		// if we find a ']', we ignore anything 2 positions from it
		// (itself, plus a blank space)
		msg = msg[i+2:]
	}

	switch b[0] {
	case 'I':
		InfoStackDepth(l.stackDepth, msg)
	case 'W':
		WarnStackDepth(l.stackDepth, msg)
	case 'E':
		ErrorStackDepth(l.stackDepth, msg)
	case 'F':
		CriticalStackDepth(l.stackDepth, msg)
	default:
		InfoStackDepth(l.stackDepth, msg)
	}

	return 0, nil
}
