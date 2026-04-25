package taskmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

const (
	defaultPageSize    = 50
	pullRequestTitle   = "Pull Request"
	managedIssuesQuery = `
query ManagedIssues($projectId: ID!, $stateIds: [ID!]!, $after: String, $first: Int!) {
  issues(
    filter: {
      project: { id: { eq: $projectId } }
      state: { id: { in: $stateIds } }
    }
    after: $after
    first: $first
  ) {
    nodes {
      id
      identifier
      title
      description
      project {
        id
      }
      state {
        id
        name
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`
	issueCommentsQuery = `
	query IssueComments($issueId: String!, $after: String, $first: Int!) {
  issue(id: $issueId) {
    comments(after: $after, first: $first) {
      nodes {
        id
        body
        createdAt
        user {
          id
          name
          displayName
          email
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`
	moveTaskMutation = `
mutation MoveTask($id: ID!, $stateId: ID!) {
  issueUpdate(id: $id, input: { stateId: $stateId }) {
    success
  }
}
`
	addCommentMutation = `
mutation AddComment($issueId: ID!, $body: String!) {
  commentCreate(input: { issueId: $issueId, body: $body }) {
    success
    comment {
      id
    }
  }
}
`
	addPRMutation = `
mutation AddPR($issueId: ID!, $url: String!, $title: String!) {
  attachmentCreate(input: { issueId: $issueId, url: $url, title: $title }) {
    success
    attachment {
      id
    }
  }
}
`
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type LinearClient struct {
	Config     config.LinearTaskManagerConfig
	HTTPClient httpDoer
	LogWriter  io.Writer
}

func NewLinearClient(cfg config.LinearTaskManagerConfig) *LinearClient {
	return &LinearClient{
		Config: cfg,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (client *LinearClient) GetTasks(ctx context.Context, projectID string, stateIDs []string) ([]Task, error) {
	logger := steplog.New(writerOrDiscard(client.LogWriter))
	logger.Infof("linear", "query managed issues project=%s state_ids=%s", projectID, strings.Join(stateIDs, ","))

	type managedIssuesResponse struct {
		Issues struct {
			Nodes    []issueNode `json:"nodes"`
			PageInfo pageInfo    `json:"pageInfo"`
		} `json:"issues"`
	}

	var (
		after string
		tasks []Task
	)

	for {
		var response managedIssuesResponse
		if err := client.doGraphQL(ctx, managedIssuesQuery, map[string]any{
			"projectId": projectID,
			"stateIds":  stateIDs,
			"after":     nullableString(after),
			"first":     defaultPageSize,
		}, &response); err != nil {
			return nil, fmt.Errorf("query managed issues: %w", err)
		}

		for _, node := range response.Issues.Nodes {
			task := Task{
				ID:          strings.TrimSpace(node.ID),
				Identifier:  strings.TrimSpace(node.Identifier),
				Title:       strings.TrimSpace(node.Title),
				Description: strings.TrimSpace(node.Description),
				ProjectID:   strings.TrimSpace(node.Project.ID),
				State: WorkflowState{
					ID:   strings.TrimSpace(node.State.ID),
					Name: strings.TrimSpace(node.State.Name),
				},
				Comments: []Comment{},
			}
			if !isManagedTask(task, projectID, stateIDs) {
				continue
			}

			comments, err := client.getTaskComments(ctx, task.ID)
			if err != nil {
				return nil, fmt.Errorf("query comments for task %s: %w", task.ID, err)
			}
			task.Comments = comments
			tasks = append(tasks, task)
		}

		if !response.Issues.PageInfo.HasNextPage {
			break
		}
		after = response.Issues.PageInfo.EndCursor
	}

	logger.Infof("linear", "fetched %d managed issues project=%s", len(tasks), projectID)
	return tasks, nil
}

func (client *LinearClient) MoveTask(ctx context.Context, taskID string, stateID string) error {
	logger := steplog.New(writerOrDiscard(client.LogWriter))
	logger.Infof("linear", "mutation move task=%s state=%s", taskID, stateID)

	type moveTaskResponse struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}

	var response moveTaskResponse
	if err := client.doGraphQL(ctx, moveTaskMutation, map[string]any{
		"id":      taskID,
		"stateId": stateID,
	}, &response); err != nil {
		return fmt.Errorf("move task: %w", err)
	}
	if !response.IssueUpdate.Success {
		return fmt.Errorf("move task: linear returned success=false")
	}

	return nil
}

func (client *LinearClient) AddComment(ctx context.Context, taskID string, body string) error {
	logger := steplog.New(writerOrDiscard(client.LogWriter))
	logger.Infof("linear", "mutation add comment task=%s", taskID)

	type addCommentResponse struct {
		CommentCreate struct {
			Success bool `json:"success"`
			Comment struct {
				ID string `json:"id"`
			} `json:"comment"`
		} `json:"commentCreate"`
	}

	var response addCommentResponse
	if err := client.doGraphQL(ctx, addCommentMutation, map[string]any{
		"issueId": taskID,
		"body":    body,
	}, &response); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	if !response.CommentCreate.Success {
		return fmt.Errorf("add comment: linear returned success=false")
	}

	return nil
}

func (client *LinearClient) AddPR(ctx context.Context, taskID string, prURL string) error {
	logger := steplog.New(writerOrDiscard(client.LogWriter))
	logger.Infof("linear", "mutation add pr task=%s pr=%s", taskID, prURL)

	type addPRResponse struct {
		AttachmentCreate struct {
			Success    bool `json:"success"`
			Attachment struct {
				ID string `json:"id"`
			} `json:"attachment"`
		} `json:"attachmentCreate"`
	}

	var response addPRResponse
	if err := client.doGraphQL(ctx, addPRMutation, map[string]any{
		"issueId": taskID,
		"url":     prURL,
		"title":   pullRequestTitle,
	}, &response); err != nil {
		return fmt.Errorf("add pr attachment: %w", err)
	}
	if !response.AttachmentCreate.Success {
		return fmt.Errorf("add pr attachment: linear returned success=false")
	}

	return nil
}

func (client *LinearClient) getTaskComments(ctx context.Context, taskID string) ([]Comment, error) {
	logger := steplog.New(writerOrDiscard(client.LogWriter))
	logger.Infof("linear", "query comments task=%s", taskID)

	type issueCommentsResponse struct {
		Issue *struct {
			Comments struct {
				Nodes    []commentNode `json:"nodes"`
				PageInfo pageInfo      `json:"pageInfo"`
			} `json:"comments"`
		} `json:"issue"`
	}

	var (
		after    string
		comments []Comment
	)

	for {
		var response issueCommentsResponse
		if err := client.doGraphQL(ctx, issueCommentsQuery, map[string]any{
			"issueId": taskID,
			"after":   nullableString(after),
			"first":   defaultPageSize,
		}, &response); err != nil {
			return nil, err
		}
		if response.Issue == nil {
			return nil, fmt.Errorf("issue %s not found", taskID)
		}

		for _, node := range response.Issue.Comments.Nodes {
			comment := Comment{
				ID:   strings.TrimSpace(node.ID),
				Body: strings.TrimSpace(node.Body),
				User: User{
					ID:          strings.TrimSpace(node.User.ID),
					Name:        strings.TrimSpace(node.User.Name),
					DisplayName: strings.TrimSpace(node.User.DisplayName),
					Email:       strings.TrimSpace(node.User.Email),
				},
			}
			if createdAt := strings.TrimSpace(node.CreatedAt); createdAt != "" {
				if parsed, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
					comment.CreatedAt = parsed
				}
			}
			comments = append(comments, comment)
		}

		if !response.Issue.Comments.PageInfo.HasNextPage {
			break
		}
		after = response.Issue.Comments.PageInfo.EndCursor
	}

	if comments == nil {
		return []Comment{}, nil
	}

	return comments, nil
}

func (client *LinearClient) doGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(graphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.Config.APIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", strings.TrimSpace(client.Config.APIToken))

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform graphql request: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read graphql response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("linear api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var graphResponse graphQLResponse
	if err := json.Unmarshal(payload, &graphResponse); err != nil {
		return fmt.Errorf("decode graphql response: %w", err)
	}
	if len(graphResponse.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", graphResponse.Errors[0].Message)
	}
	if out == nil {
		return nil
	}
	if len(graphResponse.Data) == 0 || bytes.Equal(graphResponse.Data, []byte("null")) {
		return fmt.Errorf("graphql response missing data")
	}
	if err := json.Unmarshal(graphResponse.Data, out); err != nil {
		return fmt.Errorf("decode graphql data: %w", err)
	}

	return nil
}

func isManagedTask(task Task, projectID string, stateIDs []string) bool {
	if strings.TrimSpace(task.ProjectID) != strings.TrimSpace(projectID) {
		return false
	}

	stateID := strings.TrimSpace(task.State.ID)
	for _, candidate := range stateIDs {
		if stateID == strings.TrimSpace(candidate) {
			return true
		}
	}

	return false
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type issueNode struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Project     struct {
		ID string `json:"id"`
	} `json:"project"`
	State struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
}

type commentNode struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	User      struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
	} `json:"user"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}
