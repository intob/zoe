#!/bin/bash

#
# Update AWS Route53 DNS records
# with the current IP addresses of the fly app
#

IPS_LIST=$(flyctl ips list)
IPV6=$(echo "$IPS_LIST" | awk '/v6/{print $2}')
IPV4=$(echo "$IPS_LIST" | awk '/v4/{print $2}')

echo "Updating DNS records for $RECORD_NAME with IPv4: $IPV4 and IPv6: $IPV6"

aws route53 change-resource-record-sets \
      --hosted-zone-id $AWS_HOSTED_ZONE_ID \
      --change-batch "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$RECORD_NAME\",\"Type\":\"AAAA\",\"TTL\":60,\"ResourceRecords\":[{\"Value\":\"$IPV6\"}]}}]}"

aws route53 change-resource-record-sets \
      --hosted-zone-id $AWS_HOSTED_ZONE_ID \
      --change-batch "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$RECORD_NAME\",\"Type\":\"A\",\"TTL\":60,\"ResourceRecords\":[{\"Value\":\"$IPV4\"}]}}]}"

#
# Issue TLS certificate if not already issued
#

echo "Issuing TLS certificate for $RECORD_NAME"
flyctl certs create $RECORD_NAME || true # ignore error if cert already exists