#!/bin/bash

# setup-and-run.sh - Setup and run the JobSpy MCP server
# Following the coding instructions to use camelCase for bash functions and variables

# Define local variables at the top
setupAndRun() {
    local packageManager npmVersion nodeVersion installCmd

    echo "🔍 Checking environment..."
    
    # Check if Node.js is installed
    if ! command -v node &> /dev/null; then
        echo "❌ Node.js is not installed. Please install Node.js v16 or higher."
        exit 1
    fi
    
    # Get Node.js version
    nodeVersion=$(node -v | cut -d 'v' -f 2)
    echo "✅ Node.js version: $nodeVersion"
    
    # Determine package manager to use (npm, yarn, or pnpm)
    if command -v pnpm &> /dev/null; then
        packageManager="pnpm"
    elif command -v yarn &> /dev/null; then
        packageManager="yarn"
    else
        packageManager="npm"
    fi
    
    echo "📦 Using package manager: $packageManager"
    
    # Install dependencies
    echo "📥 Installing dependencies..."
    if [ "$packageManager" = "pnpm" ]; then
        installCmd="pnpm install"
    elif [ "$packageManager" = "yarn" ]; then
        installCmd="yarn"
    else
        installCmd="npm install"
    fi
    
    $installCmd
    
    if [ $? -ne 0 ]; then
        echo "❌ Failed to install dependencies. Please check your internet connection and try again."
        exit 1
    fi
    
    echo "✅ Dependencies installed successfully."
    echo "🚀 Starting JobSpy MCP server..."
    
    # Run the server
    if [ "$packageManager" = "pnpm" ]; then
        pnpm start
    elif [ "$packageManager" = "yarn" ]; then
        yarn start
    else
        npm start
    fi
}

# Execute the main function
setupAndRun
