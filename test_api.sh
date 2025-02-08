#!/bin/bash

BASE_URL="http://localhost:8080"

echo "Creating a new note..."
curl -X POST $BASE_URL/notes \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Note",
    "content": "This is a test note content"
  }' | json_pp

echo -e "\nGetting all notes..."
curl $BASE_URL/notes | json_pp

echo -e "\nGetting note with ID 1..."
curl $BASE_URL/notes/1 | json_pp

echo -e "\nUpdating note with ID 1..."
curl -X PUT $BASE_URL/notes/1 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated Note",
    "content": "This note has been updated"
  }' | json_pp

echo -e "\nDeleting note with ID 1..."
curl -X DELETE $BASE_URL/notes/1
