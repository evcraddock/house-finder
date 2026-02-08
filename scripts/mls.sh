#!/bin/bash

# Usage: ./mls.sh "10109 Kay Rdg, Yukon, OK 73099"
# Requires: RAPIDAPI_KEY environment variable

if [ -z "$1" ]; then
  echo "Usage: ./mls.sh \"<address>\""
  echo "Example: ./mls.sh \"10109 Kay Rdg, Yukon, OK 73099\""
  exit 1
fi

if [ -z "$RAPIDAPI_KEY" ]; then
  echo "Error: RAPIDAPI_KEY environment variable is not set"
  exit 1
fi

ADDR="$1"

# Step 1: Get the property ID from autocomplete API
MPR_ID=$(curl -s -G "https://parser-external.geo.moveaws.com/suggest" \
  --data-urlencode "input=$ADDR" \
  --data "client_id=rdc-home" \
  --data "limit=1" \
  -H 'User-Agent: Mozilla/5.0' \
  | jq -r '.autocomplete[0].mpr_id')

if [ -z "$MPR_ID" ] || [ "$MPR_ID" = "null" ]; then
  echo "Error: Could not find property for address: $ADDR"
  exit 1
fi

# Step 2: Get the full realtor.com URL with M-number
RESULT=$(curl -s "https://www.realtor.com/api/v1/hulk_main_srp?client_id=rdc-x&schema=vesta" \
  -H 'User-Agent: Mozilla/5.0' \
  -H 'Content-Type: application/json' \
  --data "{\"query\": \"query { home(property_id: \\\"$MPR_ID\\\") { href property_id } }\"}")

HREF=$(echo "$RESULT" | jq -r '.data.home.href')

if [ -z "$HREF" ] || [ "$HREF" = "null" ]; then
  echo "Error: Could not get realtor.com URL for property ID: $MPR_ID"
  exit 1
fi

echo "Realtor.com URL: $HREF"

# Step 3: URL encode the href for the API call
ENCODED_URL=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$HREF', safe=''))")

# Step 4: Create a filename from the address (sanitize it)
FILENAME=$(echo "$ADDR" | sed 's/[^a-zA-Z0-9]/-/g' | sed 's/--*/-/g' | sed 's/-$//' | tr '[:upper:]' '[:lower:]').json

# Step 5: Fetch property details from RapidAPI and save to JSON
echo "Fetching property details..."
curl -s --request GET \
  --url "https://us-real-estate-listings.p.rapidapi.com/v2/property?property_url=$ENCODED_URL" \
  --header 'x-rapidapi-host: us-real-estate-listings.p.rapidapi.com' \
  --header "x-rapidapi-key: $RAPIDAPI_KEY" \
  -o "$FILENAME"

echo "Saved to: $FILENAME"
