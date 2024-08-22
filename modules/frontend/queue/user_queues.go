package queue

// This struct holds user queues for pending requests. It also keeps track of connected queriers,
// and mapping between users and queriers.
type queues struct {
	userQueues map[string]*userQueue

	// List of all users with queues, used for iteration when searching for next queue to handle.
	// Users removed from the middle are replaced with "". To avoid skipping users during iteration, we only shrink
	// this list when there are ""'s at the end of it.
	users []string

	maxUserQueueSize int
}

type userQueue struct {
	ch chan Request

	// Points back to 'users' field in queues. Enables quick cleanup.
	index int
}

func newUserQueues(maxUserQueueSize int) *queues {
	return &queues{
		userQueues:       map[string]*userQueue{},
		users:            nil,
		maxUserQueueSize: maxUserQueueSize,
	}
}

func (q *queues) len() int {
	return len(q.userQueues)
}

func (q *queues) deleteQueue(userID string) {
	uq := q.userQueues[userID]
	if uq == nil {
		return
	}

	delete(q.userQueues, userID)
	q.users[uq.index] = ""

	// Shrink users list size if possible. This is safe, and no users will be skipped during iteration.
	for ix := len(q.users) - 1; ix >= 0 && q.users[ix] == ""; ix-- {
		q.users = q.users[:ix]
	}
}

// Returns existing or new queue for user.
// MaxQueriers is used to compute which queriers should handle requests for this user.
// If maxQueriers is <= 0, all queriers can handle this user's requests.
// If maxQueriers has changed since the last call, queriers for this are recomputed.
func (q *queues) getOrAddQueue(userID string) chan Request {
	// Empty user is not allowed, as that would break our users list ("" is used for free spot).
	if userID == "" {
		return nil
	}

	uq := q.userQueues[userID]

	if uq == nil {
		uq = &userQueue{
			ch:    make(chan Request, q.maxUserQueueSize),
			index: -1,
		}
		q.userQueues[userID] = uq

		// Add user to the list of users... find first free spot, and put it there.
		for ix, u := range q.users {
			if u == "" {
				uq.index = ix
				q.users[ix] = userID
				break
			}
		}

		// ... or add to the end.
		if uq.index < 0 {
			uq.index = len(q.users)
			q.users = append(q.users, userID)
		}
	}

	return uq.ch
}

// Finds next queue for the querier. To support fair scheduling between users, client is expected
// to pass last user index returned by this function as argument. Is there was no previous
// last user index, use -1.
func (q *queues) getNextQueueForQuerier(lastUserIndex int, querierID string) (chan Request, string, int) {
	uid := lastUserIndex

	for iters := 0; iters < len(q.users); iters++ {
		uid = uid + 1

		// Don't use "mod len(q.users)", as that could skip users at the beginning of the list
		// for example when q.users has shrunk since last call.
		if uid >= len(q.users) {
			uid = 0
		}

		u := q.users[uid]
		if u == "" {
			continue
		}

		q := q.userQueues[u]

		if len(q.ch) == 0 {
			continue
		}

		return q.ch, u, uid
	}
	return nil, "", uid
}
