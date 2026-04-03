#!/bin/bash

# --- Configuration ---
# Colors for output
GREEN='\033[0;32m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${PURPLE}🚀 Starting neknow-bot development stack...${NC}"

# Function to kill all background processes on exit
cleanup() {
    echo -e "\n${YELLOW}🛑 Shutting down all services...${NC}"
    kill $(jobs -p)
    exit
}

trap cleanup SIGINT SIGTERM

# 1. Check if Infrastructure is running
if ! docker ps | grep -q "neknowbot_postgres"; then
    echo -e "${YELLOW}⚠️  Warning: Infrastructure containers (Postgres, etc.) don't seem to be running.${NC}"
    echo -e "Consider running: ${CYAN}docker-compose up -d postgres redis rabbitmq minio${NC}"
fi

# 2. Start Python Services (API + Worker)
# Note: worker/main.py starts both the FastAPI server and the consumer.
echo -e "${GREEN}🐍 Starting Python Services (API + Worker)...${NC}"
cd worker
source .venv/bin/activate
export PYTHONPATH=$PYTHONPATH:.
python main.py &
cd ..

# 3. Start Go Backend API (Port 8080)
echo -e "${GREEN}🐹 Starting Go Backend API (Port 8080)...${NC}"
go run cmd/api/main.go &

# 4. Start React Frontend (Vite)
echo -e "${GREEN}⚛️  Starting React Frontend (Port 5173)...${NC}"
cd neknow-frontend
npm run dev &
cd ..

echo -e "${PURPLE}✨ All services are requested to start. Press Ctrl+C to stop all.${NC}"

# Wait for all background processes
wait
