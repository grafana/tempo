package main

import "time"

func testPull(number int) pullRequest {
	return pullRequest{
		Number:    number,
		CreatedAt: testNow.Add(-24 * time.Hour),
		State:     "open",
		User:      &user{Login: "external", Type: "User"},
		Assignees: []user{},
		Labels:    []label{},
	}
}

func assignedTo(pr pullRequest, names ...string) pullRequest {
	for _, name := range names {
		pr.Assignees = append(pr.Assignees, user{Login: name})
	}
	return pr
}
