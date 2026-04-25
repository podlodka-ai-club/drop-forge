package taskmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLinearClientGetTasksFiltersProjectAndStateAndLoadsComments(t *testing.T) {
	httpClient := &scriptedHTTPClient{
		responses: []scriptedResponse{
			{
				body: `{"data":{"issues":{"nodes":[
					{"id":"issue-1","identifier":"ENG-1","title":"Proposal task","description":"desc","project":{"id":"project-123"},"state":{"id":"state-1","name":"Ready to Propose"}},
					{"id":"issue-2","identifier":"ENG-2","title":"Other project","description":"desc","project":{"id":"project-999"},"state":{"id":"state-1","name":"Ready to Propose"}},
					{"id":"issue-3","identifier":"ENG-3","title":"Other state","description":"desc","project":{"id":"project-123"},"state":{"id":"state-3","name":"Backlog"}},
					{"id":"issue-4","identifier":"ENG-4","title":"No comments","description":"","project":{"id":"project-123"},"state":{"id":"state-2","name":"Ready to Code"}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`,
			},
			{
				body: `{"data":{"issue":{"comments":{"nodes":[
					{"id":"comment-1","body":"Need revision","createdAt":"2026-04-25T09:00:00Z","user":{"id":"user-1","name":"Alex","displayName":"Alex","email":"alex@example.com"}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
			},
			{
				body: `{"data":{"issue":{"comments":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
			},
		},
	}

	var logs bytes.Buffer
	client := &LinearClient{
		Config:     validConfig(),
		HTTPClient: httpClient,
		LogWriter:  &logs,
	}

	tasks, err := client.GetTasks(context.Background(), "project-123", []string{"state-1", "state-2"})
	if err != nil {
		t.Fatalf("GetTasks() returned error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("tasks len = %d, want 2", len(tasks))
	}
	if tasks[0].ID != "issue-1" || tasks[1].ID != "issue-4" {
		t.Fatalf("task ids = %#v", []string{tasks[0].ID, tasks[1].ID})
	}
	if tasks[0].Comments[0].Body != "Need revision" {
		t.Fatalf("first task comments = %#v", tasks[0].Comments)
	}
	if tasks[1].Description != "" {
		t.Fatalf("second task description = %q, want empty string", tasks[1].Description)
	}
	if tasks[1].Comments == nil || len(tasks[1].Comments) != 0 {
		t.Fatalf("second task comments = %#v, want empty slice", tasks[1].Comments)
	}

	if len(httpClient.requests) != 3 {
		t.Fatalf("requests len = %d, want 3", len(httpClient.requests))
	}

	firstReq := httpClient.requests[0]
	if got := firstReq.Headers.Get("Authorization"); got != "linear-token" {
		t.Fatalf("Authorization header = %q, want %q", got, "linear-token")
	}
	if got := firstReq.Headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := firstReq.Variables["projectId"]; got != "project-123" {
		t.Fatalf("projectId variable = %#v", got)
	}
	assertStringSliceVariable(t, firstReq.Variables["stateIds"], []string{"state-1", "state-2"})

	events := decodeTaskManagerEvents(t, logs.String())
	assertTaskManagerLog(t, events, "linear", "query managed issues project=project-123 state_ids=state-1,state-2")
	assertTaskManagerLog(t, events, "linear", "query comments task=issue-1")
	assertTaskManagerLog(t, events, "linear", "fetched 2 managed issues project=project-123")
}

func TestLinearClientGetTasksReturnsUpdatedCommentsOnRepeatedFetch(t *testing.T) {
	httpClient := &scriptedHTTPClient{
		responses: []scriptedResponse{
			{
				body: `{"data":{"issues":{"nodes":[
					{"id":"issue-1","identifier":"ENG-1","title":"Proposal task","description":"desc","project":{"id":"project-123"},"state":{"id":"state-1","name":"Ready to Propose"}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`,
			},
			{
				body: `{"data":{"issue":{"comments":{"nodes":[
					{"id":"comment-1","body":"First feedback","createdAt":"2026-04-25T09:00:00Z","user":{"id":"user-1","name":"Alex","displayName":"Alex","email":"alex@example.com"}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
			},
			{
				body: `{"data":{"issues":{"nodes":[
					{"id":"issue-1","identifier":"ENG-1","title":"Proposal task","description":"desc","project":{"id":"project-123"},"state":{"id":"state-1","name":"Ready to Propose"}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`,
			},
			{
				body: `{"data":{"issue":{"comments":{"nodes":[
					{"id":"comment-1","body":"First feedback","createdAt":"2026-04-25T09:00:00Z","user":{"id":"user-1","name":"Alex","displayName":"Alex","email":"alex@example.com"}},
					{"id":"comment-2","body":"Please fix title","createdAt":"2026-04-25T10:00:00Z","user":{"id":"user-2","name":"Dana","displayName":"Dana","email":"dana@example.com"}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
			},
		},
	}
	client := &LinearClient{
		Config:     validConfig(),
		HTTPClient: httpClient,
	}

	first, err := client.GetTasks(context.Background(), "project-123", []string{"state-1"})
	if err != nil {
		t.Fatalf("first GetTasks() error = %v", err)
	}
	second, err := client.GetTasks(context.Background(), "project-123", []string{"state-1"})
	if err != nil {
		t.Fatalf("second GetTasks() error = %v", err)
	}

	if len(first) != 1 || len(first[0].Comments) != 1 {
		t.Fatalf("first comments = %#v", first)
	}
	if len(second) != 1 || len(second[0].Comments) != 2 {
		t.Fatalf("second comments = %#v", second)
	}
	if second[0].Comments[1].Body != "Please fix title" {
		t.Fatalf("updated comment = %#v", second[0].Comments)
	}
}

func TestLinearClientWriteOperationsSendExpectedRequests(t *testing.T) {
	tests := []struct {
		name         string
		call         func(context.Context, *LinearClient) error
		responseBody string
		wantQuery    string
		wantVars     map[string]any
	}{
		{
			name: "move task",
			call: func(ctx context.Context, client *LinearClient) error {
				return client.MoveTask(ctx, "issue-1", "state-2")
			},
			responseBody: `{"data":{"issueUpdate":{"success":true}}}`,
			wantQuery:    "issueUpdate",
			wantVars: map[string]any{
				"id":      "issue-1",
				"stateId": "state-2",
			},
		},
		{
			name: "add comment",
			call: func(ctx context.Context, client *LinearClient) error {
				return client.AddComment(ctx, "issue-1", "hello")
			},
			responseBody: `{"data":{"commentCreate":{"success":true,"comment":{"id":"comment-1"}}}}`,
			wantQuery:    "commentCreate",
			wantVars: map[string]any{
				"issueId": "issue-1",
				"body":    "hello",
			},
		},
		{
			name: "add pr",
			call: func(ctx context.Context, client *LinearClient) error {
				return client.AddPR(ctx, "issue-1", "https://github.com/example/project/pull/42")
			},
			responseBody: `{"data":{"attachmentCreate":{"success":true,"attachment":{"id":"attachment-1"}}}}`,
			wantQuery:    "attachmentCreate",
			wantVars: map[string]any{
				"issueId": "issue-1",
				"url":     "https://github.com/example/project/pull/42",
				"title":   "Pull Request",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &scriptedHTTPClient{
				responses: []scriptedResponse{{body: tt.responseBody}},
			}
			client := &LinearClient{
				Config:     validConfig(),
				HTTPClient: httpClient,
			}

			if err := tt.call(context.Background(), client); err != nil {
				t.Fatalf("call returned error: %v", err)
			}
			if len(httpClient.requests) != 1 {
				t.Fatalf("requests len = %d, want 1", len(httpClient.requests))
			}

			req := httpClient.requests[0]
			if !strings.Contains(req.Query, tt.wantQuery) {
				t.Fatalf("query = %q, want substring %q", req.Query, tt.wantQuery)
			}
			for key, want := range tt.wantVars {
				if got := req.Variables[key]; got != want {
					t.Fatalf("variable %s = %#v, want %#v", key, got, want)
				}
			}
		})
	}
}

func TestLinearClientHandlesTransportAndGraphQLErrors(t *testing.T) {
	client := &LinearClient{
		Config: validConfig(),
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp timeout")
		}),
	}

	if _, err := client.GetTasks(context.Background(), "project-123", []string{"state-1"}); err == nil || !strings.Contains(err.Error(), "perform graphql request") {
		t.Fatalf("transport error = %v, want wrapped transport error", err)
	}

	httpClient := &scriptedHTTPClient{
		responses: []scriptedResponse{{
			body: `{"errors":[{"message":"forbidden"}]}`,
		}},
	}
	client = &LinearClient{
		Config:     validConfig(),
		HTTPClient: httpClient,
	}

	err := client.MoveTask(context.Background(), "issue-1", "state-2")
	if err == nil || !strings.Contains(err.Error(), "graphql error: forbidden") {
		t.Fatalf("graphql error = %v, want wrapped graphql error", err)
	}
}

func TestLinearClientHandlesHTTPStatusAndPartialPayloads(t *testing.T) {
	httpClient := &scriptedHTTPClient{
		responses: []scriptedResponse{{
			statusCode: http.StatusBadGateway,
			body:       `upstream failed`,
		}},
	}
	client := &LinearClient{
		Config:     validConfig(),
		HTTPClient: httpClient,
	}

	if _, err := client.GetTasks(context.Background(), "project-123", []string{"state-1"}); err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("status error = %v, want wrapped status error", err)
	}

	httpClient = &scriptedHTTPClient{
		responses: []scriptedResponse{
			{
				body: `{"data":{"issues":{"nodes":[
					{"id":"issue-1","identifier":"ENG-1","title":"Task","description":"","project":{"id":"project-123"},"state":{"id":"state-1","name":"Ready to Propose"}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`,
			},
			{
				body: `{"data":{"issue":{"comments":{"nodes":[
					{"id":"comment-1","body":"body","createdAt":"","user":{"id":"","name":"","displayName":"","email":""}}
				],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
			},
		},
	}
	client = &LinearClient{
		Config:     validConfig(),
		HTTPClient: httpClient,
	}

	tasks, err := client.GetTasks(context.Background(), "project-123", []string{"state-1"})
	if err != nil {
		t.Fatalf("GetTasks() returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks len = %d, want 1", len(tasks))
	}
	if tasks[0].Description != "" {
		t.Fatalf("description = %q, want empty", tasks[0].Description)
	}
	if tasks[0].Comments[0].CreatedAt.String() != "0001-01-01 00:00:00 +0000 UTC" {
		t.Fatalf("CreatedAt = %v, want zero time", tasks[0].Comments[0].CreatedAt)
	}
}

type scriptedHTTPClient struct {
	requests  []capturedRequest
	responses []scriptedResponse
	index     int
}

type scriptedResponse struct {
	statusCode int
	body       string
	err        error
}

type capturedRequest struct {
	Query     string
	Variables map[string]any
	Headers   http.Header
}

func (client *scriptedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if client.index >= len(client.responses) {
		return nil, fmt.Errorf("unexpected request %d", client.index+1)
	}

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var graphQLReq graphQLRequest
	if err := json.Unmarshal(payload, &graphQLReq); err != nil {
		return nil, err
	}
	client.requests = append(client.requests, capturedRequest{
		Query:     graphQLReq.Query,
		Variables: graphQLReq.Variables,
		Headers:   req.Header.Clone(),
	})

	response := client.responses[client.index]
	client.index++
	if response.err != nil {
		return nil, response.err
	}

	statusCode := response.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(response.body)),
		Header:     make(http.Header),
	}, nil
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func assertStringSliceVariable(t *testing.T, got any, want []string) {
	t.Helper()

	items, ok := got.([]any)
	if !ok {
		t.Fatalf("variable type = %T, want []any", got)
	}
	if len(items) != len(want) {
		t.Fatalf("variable len = %d, want %d", len(items), len(want))
	}
	for index, item := range items {
		if item != want[index] {
			t.Fatalf("variable[%d] = %#v, want %#v", index, item, want[index])
		}
	}
}
