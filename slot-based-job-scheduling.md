# Slot-based job scheduling for the read path

Grafana Tempo · Read path · Proposal

Query latency is ultimately set by the slowest required job.
Today's batch protocol adds another barrier:
completed jobs are not returned,
and their stream cannot fetch replacement work,
until every job in the batch finishes.
This proposal makes the job the real scheduling unit while retaining coalesced transport.

## The problem

The frontend shards a query into jobs and hands them to queriers in batches of
`max_batch_size`.
Each Process stream receives one batch at a time.
The querier executes the jobs concurrently but returns the batch only when every job in it finishes.
When one job is slow,
completed sibling results remain buffered and that stream cannot request replacement work,
even if the frontend still has unassigned jobs.
This can strand dispatch capacity and contribute to the hot-spot / idle-capacity pattern we see in production.

Batches are also heterogeneous.
The frontend fills a batch from the tenant's queue, not from a single query,
so when queries A and B arrive concurrently a batch can look like `[A1, B3, B7]`.
Query B's completed jobs then wait behind query A's straggler,
and a slow job in one query lengthens the tail of unrelated queries
that happened to share its batches.
The barrier couples not just siblings within a query,
but concurrent queries of the same tenant.

This proposal does not make an already-running straggler stealable.
If only in-flight stragglers remain,
other queriers will still be idle;
reducing that final tail requires finer jobs, splitting, or speculative execution.

Today each job is a fixed HTTP subrequest, queue item, and cache fragment,
while the Process protocol assigns and completes a group of jobs atomically.
That couples transport batching to ownership and completion.
Making jobs finer increases per-job overhead across queries with hundreds of thousands of jobs,
but retaining larger transport batches also retains the completion barrier.
The proposal separates those concerns.

## The proposal

Make the job the unit of execution and capacity accounting.
Batches survive as a wire-level optimisation,
but stop being an ownership and completion barrier.
Completed jobs return in coalesced frames and release their slots
without waiting for every sibling from the original assignment.

### Illustrative simulation

- 36 jobs, three queriers, four execution slots per querier (12 total),
  and batches of four jobs in batch mode.
- Most jobs take 330–720 ms;
  job 3 takes 2,000 ms and job 18 takes 1,800 ms.
- Batch mode buffers completed results until the entire batch finishes,
  then returns all results and allows the stream to fetch another batch.
- Slot mode buffers completed results until the next 200 ms frame exchange,
  then returns those results and assigns replacement work to free slots.
  There is no prefetch reserve in this example.

Same query, same three queriers, and the same aggregate job concurrency.
The model uses one active batch-processing lane per querier for clarity;
in Tempo, a querier normally has multiple Process streams,
and a straggler blocks one stream from fetching replacement work,
not the whole querier.
In the proposal, completed slots are refilled on each 200 ms frame exchange.
Already-running stragglers remain assigned.

The proposal has two mechanisms:
make querier capacity explicit,
and decouple transport batching from job completion and slot reuse.

### 1. Make executable capacity an explicit contract

Each querier gets a hard, configured `max_concurrent_jobs`,
where one running job occupies one slot.
Today the upper bound is up to the number of active Process streams multiplied by
`max_batch_size`;
request weights generally reduce the actual number of jobs.
The slot limit provides a clear safety invariant,
but it must be paired with credit-aware dispatch
so excess work does not become hidden backlog inside the querier.

### 2. Refill slots independently; coalesce protocol messages

Assignments and results still travel in coalesced frames.
A completed job releases its slot in the next completion frame
without waiting for sibling jobs from its original assignment.
Since batches mix queries,
this also stops one query's straggler from holding back another query's completed results.
Frames flush by count, bytes, or maximum delay,
so capacity is not reported after every job.
Stable query-execution and job identifiers let the frontend track outstanding work
and match out-of-order results.
A running straggler remains assigned;
this mechanism recovers capacity from completed siblings rather than stealing in-flight work.

### Design principle

The scheduler stays boring.
Capacity is configured, not learned,
and dispatch decisions are reconstructable from a small set of metrics:
configured, executing, and free slots;
unassigned queue depth;
and job durations.
These signals isolate the scheduler's contribution to a slow query
without hiding decisions behind an adaptive capacity model.

## Open questions

- When queriers are idle,
  does the frontend still have unassigned work,
  or are only in-flight stragglers left?
  The former supports this proposal;
  the latter requires finer work or speculation.
- What completion count, byte limit, and maximum delay keep frame traffic scalable
  without adding meaningful refill latency?
- What `max_concurrent_jobs` value saturates a querier
  without causing CPU throttling, memory pressure, or excessive storage concurrency?

From a design session on `grafana/tempo`, 27 August 2026 · mapno
