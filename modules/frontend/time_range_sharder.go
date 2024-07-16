package frontend

// sharder that takes an interface? or functions to provide a generic way to shard a given query over
// a time range. we will execute queries in reverse time order starting from the end and progressing
// toward the start. every time a query finishes we will check the combiner to see if the query is
// complete. if it is then we will sort and return the results

// add a local interface for a "combinersorter" to sort results

// todo: jpe
//  - change "additional data" to "request data" or "request context" - sure
//  - execute one request for each hour working backwards
//    - 2 simultaneous requests
//  - combiner holds the N most recent results
//  - after each batch check if the combiner has enough to return
//  - adjust the sharder to be exclusive on either the front or end of the range
//
// This causes overscan of the blocks! (unless you pass both "real" range and sub "range" the real range is passed to the queriers and the sub range is used to select the blocks)
//   can we just watch the jobs come in record what the closest to now "in flight" job is? and only return if the closest to now in flight job's block does not overlap w/ the oldest sorted result?
