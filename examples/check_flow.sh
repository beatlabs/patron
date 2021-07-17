#!/bin/bash

# This script checks that after having run 'start_processing.sh' against the example services
# the message went through every single service in the chain. This is used to make sure that
# future enhancements in the product won't break the examples

# Check that http received calls and issued 200 code responses
successfulHandles=$(curl -m 1 --request GET 'http://localhost:50000/metrics' \
  | grep 'promhttp_metric_handler_requests_total{' \
  | grep '200' \
  | cut -f2 -d' ')
if (( successfulHandles < 1 )); then
  echo "http-svc did not handle at least one successful request"
  exit 1
fi

# Check that http calls http-sec
httpSvcCallsToHttpSec=$(curl -m 1 --request GET 'http://localhost:50000/metrics' \
  | grep 'client_http_request_duration_seconds_count' \
  | grep 'localhost:50001' \
  | cut -f2 -d' ')
if (( httpSvcCallsToHttpSec < 1 )); then
  echo "No client call performed between http-svc and http-sec-svc"
  exit 1
fi

# Check that http calls http-cache
httpSvcCallsToHttpcache=$(curl -m 1 --request GET 'http://localhost:50000/metrics' \
  | grep 'client_http_request_duration_seconds_count' \
  | grep 'localhost:50007' \
  | cut -f2 -d' ')
if (( httpSvcCallsToHttpcache < 1 )); then
  echo "No client call performed between http-svc and http-cache-svc"
  exit 1
fi

exit 0