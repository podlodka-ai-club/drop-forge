// Package taskmanager provides a Linear-facing integration layer for the future
// CoreOrch component. It is responsible for reading managed tasks from one
// configured Linear project and writing back task updates such as state
// transitions, comments, and PR links. The package does not implement polling,
// scheduling, or executor dispatch.
package taskmanager
