#!/bin/bash

# Test runner script for go-p2p package

echo "Running Go P2P Test Suite"
echo "========================"

# Run all tests with coverage
echo "Running unit tests with coverage..."
go test -v -race -coverprofile=coverage.out ./...

# Run benchmarks (optional)
if [ "$1" == "bench" ]; then
    echo ""
    echo "Running benchmarks..."
    go test -bench=. -benchmem ./...
fi

# Run integration tests (optional)
if [ "$1" == "integration" ] || [ "$1" == "all" ]; then
    echo ""
    echo "Running integration tests..."
    go test -v -tags=integration -run="TestIntegration" ./...
fi

# Generate coverage report
echo ""
echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "Test execution complete!"
echo "Coverage report generated: coverage.html"
