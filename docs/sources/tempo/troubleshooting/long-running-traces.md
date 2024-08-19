---
title: Long-running traces
description: Troubleshoot search results when using long-running traces
weight: 479
aliases:
  - ../operations/troubleshooting/long-running-traces/
---

# Long-running traces

An issue arises in Tempo when a user exercises a usage pattern referred to as
long-running traces. This happens is when Tempo spans receives spans for a
trace, and then there is a delay, and then Tempo receives additional spans for
the same trace. If the delay between spans is great enough, the spans end up in
different blocks, which can lead to inconsistency in a few ways.

1. When using TraceQL search, the duration information only pertains to the
   information contained in a single block. This happens because Tempo consults
   only the first matching block to determine this information. When searching a full
   trace by ID, all blocks are searched for parts of this trace,
   and so the information is more accurate.

1. When using structural operators, the conditions may match on different
   blocks, and so results can be confusing.
