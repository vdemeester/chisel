#!/bin/bash
# Helper script to create a test Tekton bundle locally for testing the bundles resolver
#
# Prerequisites:
# - tkn CLI installed (https://tekton.dev/docs/cli/)
# - Docker or Podman running
# - Access to a container registry
#
# Usage:
#   ./create-test-bundle.sh <registry> <tag>
#
# Example:
#   ./create-test-bundle.sh localhost:5000 v1
#   ./create-test-bundle.sh ghcr.io/yourname v1.0.0

set -e

REGISTRY=${1:-localhost:5000}
TAG=${2:-v1}

echo "Creating test Tekton bundle..."
echo "Registry: $REGISTRY"
echo "Tag: $TAG"

# Create a simple test task
cat > /tmp/test-task.yaml <<EOF
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: hello
spec:
  params:
  - name: message
    type: string
    default: "Hello from bundle!"
  steps:
  - name: echo
    image: alpine
    script: |
      echo "\$(params.message)"
EOF

# Check if tkn is installed
if ! command -v tkn &> /dev/null; then
    echo "Error: tkn CLI not found. Install from https://tekton.dev/docs/cli/"
    echo ""
    echo "Alternative: Use docker/podman directly to create OCI image"
    exit 1
fi

# Push the bundle
BUNDLE_REF="${REGISTRY}/tekton/hello:${TAG}"
echo ""
echo "Pushing bundle to ${BUNDLE_REF}..."
tkn bundle push "${BUNDLE_REF}" -f /tmp/test-task.yaml

echo ""
echo "✅ Bundle created successfully!"
echo ""
echo "Test it with:"
echo "  chisel run - <<EOF"
echo "  apiVersion: tekton.dev/v1"
echo "  kind: PipelineRun"
echo "  metadata:"
echo "    name: test-bundle"
echo "  spec:"
echo "    pipelineSpec:"
echo "      tasks:"
echo "      - name: hello-task"
echo "        taskRef:"
echo "          resolver: bundles"
echo "          params:"
echo "          - name: bundle"
echo "            value: ${BUNDLE_REF}"
echo "          - name: name"
echo "            value: hello"
echo "        params:"
echo "        - name: message"
echo "          value: 'Testing bundles resolver!'"
echo "  EOF"
echo ""

# Cleanup
rm /tmp/test-task.yaml
