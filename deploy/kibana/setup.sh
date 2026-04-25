#!/bin/sh
set -e

until curl -sf http://kibana:5601/api/status > /dev/null; do
  echo "waiting for kibana..."
  sleep 2
done

echo "importing saved objects..."
curl -sf -X POST "http://kibana:5601/api/saved_objects/_import?overwrite=true" \
  -H "kbn-xsrf: true" \
  --form file=@/saved-objects.ndjson

echo "kibana setup complete"
