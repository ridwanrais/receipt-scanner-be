#!/bin/bash

# Development script with Swagger auto-regeneration
# This script uses wgo for hot reload and fswatch for Swagger regeneration

echo "🚀 Starting development server with Swagger auto-regeneration..."
echo ""

# Initial Swagger generation
echo "📝 Generating initial Swagger documentation..."
~/go/bin/swag init -g cmd/server/main.go -o docs 2>&1 | grep -v "warning: failed to get package name"
echo "✓ Swagger docs generated"
echo ""

# Function to regenerate Swagger
regenerate_swagger() {
    while true; do
        # Wait a bit to debounce rapid file changes
        sleep 0.5
        
        # Regenerate Swagger
        ~/go/bin/swag init -g cmd/server/main.go -o docs 2>&1 | grep -v "warning: failed to get package name" > /dev/null
        
        # Only print if successful
        if [ $? -eq 0 ]; then
            echo "✓ Swagger regenerated at $(date '+%H:%M:%S')"
        fi
    done
}

# Check if fswatch is available
if command -v fswatch &> /dev/null; then
    # Start Swagger watcher in background
    (
        fswatch -o \
            --exclude='docs/' \
            --exclude='bin/' \
            --exclude='tmp/' \
            --exclude='vendor/' \
            --exclude='_test\.go$' \
            internal/handler/ internal/domain/ internal/model/ cmd/server/main.go | \
        while read; do
            ~/go/bin/swag init -g cmd/server/main.go -o docs 2>&1 | grep -v "warning: failed to get package name" > /dev/null
            echo "✓ Swagger regenerated at $(date '+%H:%M:%S')"
        done
    ) &
    FSWATCH_PID=$!
    
    # Cleanup function
    cleanup() {
        echo ""
        echo "🛑 Stopping Swagger watcher..."
        kill $FSWATCH_PID 2>/dev/null
        exit 0
    }
    
    trap cleanup INT TERM
    
    echo "👀 Watching for changes to regenerate Swagger..."
    echo ""
fi

# Start wgo for hot reload
~/go/bin/wgo \
    -file=.go \
    -xfile=_test.go \
    -xdir=tmp,bin,vendor,.git,docs \
    go run cmd/server/main.go

# Cleanup on exit
if [ ! -z "$FSWATCH_PID" ]; then
    kill $FSWATCH_PID 2>/dev/null
fi
