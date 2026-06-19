// Package tools holds repo-level go:generate directives. It is not a runnable
// binary; the application entrypoint is cmd/app/main.go.
package tools

//go:generate go run github.com/swaggo/swag/cmd/swag init -g cmd/app/main.go
//go:generate go run github.com/google/wire/cmd/wire ./di
