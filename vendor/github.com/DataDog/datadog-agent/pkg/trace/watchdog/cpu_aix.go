// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package watchdog

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

// From proc(5) on AIX 7.2
//        status
//             Contains state information about the process and one of its
//             representative thread. The file is formatted as a struct pstatus
//             type containing the following members:
//
//             uint32_t pr_flag;                   /* process flags from proc struct p_flag */
//             uint32_t pr_flag2;                  /* process flags from proc struct p_flag2 */
//             uint32_t pr_flags;                  /* /proc flags */
//             uint32_t pr_nlwp;                   /* number of threads in the process */
//             char     pr_stat;                   /* process state from proc p_stat */
//             char     pr_dmodel;                 /* data model for the process */
//             char     pr__pad1[6];               /* reserved for future use */
//             pr_sigset_t pr_sigpend;             /* set of process pending signals */
//             prptr64_t pr_brkbase;               /* address of the process heap */
//             uint64_t pr_brksize;                /* size of the process heap, in bytes */
//             prptr64_t pr_stkbase;               /* address of the process stack */
//             uint64_t pr_stksize;                /* size of the process stack, in bytes */
//             pid64_t  pr_pid;                    /* process id */
//             pid64_t  pr_ppid;                   /* parent process id */
//             pid64_t  pr_pgid;                   /* process group id */
//             pid64_t  pr_sid;                    /* session id */
//             struct pr_timestruc64_t pr_utime;   /* process user cpu time */
//             struct pr_timestruc64_t pr_stime;   /* process system cpu time */
//             struct pr_timestruc64_t pr_cutime;  /* sum of children's user times */
//             struct pr_timestruc64_t pr_cstime;  /* sum of children's system times */
//             pr_sigset_t pr_sigtrace;            /* mask of traced signals */
//             fltset_t pr_flttrace;               /* mask of traced hardware faults */
//             uint32_t pr_sysentry_offset;        /* offset into pstatus file of sysset_t
//                                                  * identifying system calls traced on
//
//                                                  * entry.  If 0, then no entry syscalls
//                                                  * are being traced. */
//             uint32_t pr_sysexit_offset;         /* offset into pstatus file of sysset_t
//                                                  * identifying system calls traced on
//                                                  * exit.  If 0, then no exit syscalls
//                                                  * are being traced. */
//             uint64_t pr__pad[8];                /* reserved for future use */
//             lwpstatus_t pr_lwp;                 /* "representative" thread status */
//
// From /usr/include/sys/procfs.h
// typedef struct pr_sigset
// {
//    uint64_t ss_set[4];              /* signal set */
// } pr_sigset_t;
//
// typedef struct pr_timestruc64
// {
//    int64_t  tv_sec;                 /* 64 bit time_t value      */
//    int32_t  tv_nsec;                /* 32 bit suseconds_t value */
//    uint32_t __pad;                  /* reserved for future use  */
// } pr_timestruc64_t;
//
// typedef void * prptr64_t;
//
// The fields before the user cpu time (pr_utime) are:
//	uint32_t pr_flag;		4				4
//	uint32_t pr_flag2;		4				8
//	uint32_t pr_flags;		4				12
//	uint32_t pr_nlwp;		4				16
//	char     pr_stat;		1				17
//	char     pr_dmodel;		1				18
//	char     pr__pad1[6];	6				24
//	pr_sigset_t pr_sigpend;	(4 * 8) = 32	56
//	prptr64_t pr_brkbase;	8				64
//	uint64_t pr_brksize;	8				72
//	prptr64_t pr_stkbase;	8				80
//	uint64_t pr_stksize;	8				88
//	pid64_t  pr_pid;		8				96
//	pid64_t  pr_ppid;		8				104
//	pid64_t  pr_pgid;		8				112
//	pid64_t  pr_sid;		8				120
//  total:                  120
// followed by:
//	struct pr_timestruc64_t pr_utime;   /* process user cpu time */

func cpuTimeUser(pid int32) (float64, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	// As explained above, we will skip 120 bytes into the status file to locate the user CPU time.
	f.Seek(120, os.SEEK_SET)
	var (
		userSecs  int64
		userNsecs int32
	)
	binary.Read(f, binary.BigEndian, &userSecs)
	binary.Read(f, binary.BigEndian, &userNsecs)
	time := float64(userSecs) + (float64(userNsecs) / float64(time.Second))
	return time, nil
}

func getpid() int {
	return os.Getpid()
}
