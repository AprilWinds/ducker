#!/bin/sh

echo "================================"
echo "  Ducker Test Container"
echo "================================"
echo ""
echo "APP_NAME: $APP_NAME"
echo "APP_VERSION: $APP_VERSION"
echo "WORKDIR: $(pwd)"
echo ""
echo "Environment variables:"
env | grep -E "^APP_|^HOME|^PATH" | sort
echo ""
echo "Container is running!"
echo "Sleeping..."
while true; do sleep 3600; done
