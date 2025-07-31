package jira

// SearchResult represents the top-level structure from the JIRA search API
type SearchResult struct {
	Issues []Issue `json:"issues"`
}

// Issue represents a single issue in the search result
type Issue struct {
	Key    string `json:"key"`
	Fields Fields `json:"fields"`
}

// Fields represents the inner fields of a JIRA issue
type Fields struct {
	Summary  string `json:"summary"`
	Status   Status `json:"status"`
	Assignee *User  `json:"assignee"` // nullable
}

// Status represents the status field of the issue
type Status struct {
	Name string `json:"name"`
}

// User represents the assignee or reporter of the issue
type User struct {
	DisplayName string `json:"displayName"`
}
