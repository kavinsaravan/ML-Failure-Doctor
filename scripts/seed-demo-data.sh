#!/bin/bash

API_URL="https://invigorating-empathy-production-dee5.up.railway.app"

echo "Seeding demo data to CrashLens..."

# Run various test jobs
echo "1. Running GPU OOM test..."
curl -X POST $API_URL/workloads/run \
  -H "Content-Type: application/json" \
  -d '{"template": "gpu_oom", "type": "ML_JOB"}' \
  -s > /dev/null

sleep 6

echo "2. Running successful job..."
curl -X POST $API_URL/workloads/run \
  -H "Content-Type: application/json" \
  -d '{"template": "successful", "type": "ML_JOB"}' \
  -s > /dev/null

sleep 6

echo "3. Running dependency error..."
curl -X POST $API_URL/workloads/run \
  -H "Content-Type: application/json" \
  -d '{"template": "dependency_error", "type": "ML_JOB"}' \
  -s > /dev/null

sleep 6

echo "4. Running timeout simulation..."
curl -X POST $API_URL/workloads/run \
  -H "Content-Type: application/json" \
  -d '{"template": "timeout", "type": "ML_JOB"}' \
  -s > /dev/null

sleep 6

echo "5. Running missing checkpoint..."
curl -X POST $API_URL/workloads/run \
  -H "Content-Type: application/json" \
  -d '{"template": "missing_checkpoint", "type": "ML_JOB"}' \
  -s > /dev/null

echo ""
echo "Demo data seeded! Check your dashboard at:"
echo "https://frontend-zeta-eight-92.vercel.app/dashboard"
