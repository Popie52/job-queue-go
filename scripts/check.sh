#!/bin/bash

# Total jobs
TOTAL=1000
# Concurrency
CONCURRENCY=20

# Base URL
URL="http://localhost:8080/submit"

# Start timer
START=$(date +%s)

# Function to submit a single job
submit_job() {
  INDEX=$1
  curl -s -X POST "$URL" \
       -H "Content-Type: application/json" \
       -d "{
            \"id\": \"job-$INDEX\",
            \"priority\": $((RANDOM % 10 + 1)),
            \"payload\": {
                \"type\": \"email\",
                \"to\": \"user$INDEX@example.com\",
                \"message\": \"Hello from job $INDEX\"
            },
            \"maxRetries\": 3
       }" > /dev/null
}

# Export function for xargs
export -f submit_job
export URL

# Generate sequence and run jobs with concurrency
seq 1 $TOTAL | xargs -n1 -P $CONCURRENCY -I{} bash -c 'submit_job "$@"' _ {}

# End timer
END=$(date +%s)
DURATION=$((END - START))
echo "Total duration: $DURATION seconds"
