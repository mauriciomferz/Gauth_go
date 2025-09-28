#!/bin/bash

# Navigate to frontend directory
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go/gauth-demo-app/web/frontend

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Start the React development server
echo "Starting React development server on http://localhost:3000"
npm start
