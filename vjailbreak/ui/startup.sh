#!/bin/sh

# Use envsubst to substitute environment variables in the nginx configuration
VITE_API_TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token) envsubst < /usr/share/nginx/html/index.html > /usr/share/nginx/html/index.html.temp
mv -f /usr/share/nginx/html/index.html.temp /usr/share/nginx/html/index.html

# Start OpenResty
/usr/local/openresty/bin/openresty -g 'daemon off;'