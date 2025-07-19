// Package resource provides the core interfaces and types for resource management
// in the Cutepod reconciliation system.
//
// This package defines:
//   - Resource interface that all managed resources must implement
//   - ResourceManager interface for managing specific resource types
//   - ReconciliationError for structured error handling
//   - Common types and utilities for resource management
//
// The resource system is designed to be extensible, allowing new resource types
// to be added by implementing the Resource and ResourceManager interfaces.
package resource