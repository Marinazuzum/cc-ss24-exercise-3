#!/bin/bash
# Push all custom bookstore images to Docker Hub

docker push marinazhdanova/bookstore-api_get_books:latest
docker push marinazhdanova/bookstore-api_post_books:latest
docker push marinazhdanova/bookstore-api_put_books:latest
docker push marinazhdanova/bookstore-api_delete_books:latest
docker push marinazhdanova/bookstore-frontend_renderer:latest

echo "All images pushed to Docker Hub."
