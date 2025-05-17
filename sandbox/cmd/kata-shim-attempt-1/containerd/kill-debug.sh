#!/bin/bash
# This script kills any running containerd debug processes

# Find and kill the debug processes
pids=$(ps aux | grep -E 'containerd|__debug_bin' | grep -v grep | awk '{print $2}')

if [ -z "$pids" ]; then
	echo "No containerd or debug processes found."
	exit 0
fi

echo "Found containerd processes: $pids"
echo "Sending SIGTERM..."

for pid in $pids; do
	kill -15 $pid 2> /dev/null || echo "Failed to terminate $pid"
done

# Wait a moment
sleep 2

# Check if processes are still running
still_running=""
for pid in $pids; do
	if kill -0 $pid 2> /dev/null; then
		still_running="$still_running $pid"
	fi
done

# Force kill if necessary
if [ -n "$still_running" ]; then
	echo "Some processes still running: $still_running"
	echo "Force killing with SIGKILL..."

	for pid in $still_running; do
		kill -9 $pid 2> /dev/null || echo "Failed to kill $pid"
	done
fi

echo "Done."
