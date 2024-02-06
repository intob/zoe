#!/bin/bash

#
# Update AWS Route53 DNS records for lstn.swissinfo.ch
#

IPS_LIST=$(flyctl ips list)
IPV6=$(echo "$IPS_LIST" | awk '/v6/{print $2}')
IPV4=$(echo "$IPS_LIST" | awk '/v4/{print $2}')

echo "Updating DNS records for $RECORD_NAME in hosted zone $AWS_HOSTED_ZONE_ID"
echo "IPv4: $IPV4 / IPv6: $IPV6"

# Check if records exist
AAAA_RECORDS=$(aws route53 list-resource-record-sets --hosted-zone-id $AWS_HOSTED_ZONE_ID --query "ResourceRecordSets[?Type == 'AAAA'] | [?contains(Name, '$RECORD_NAME')]")
A_RECORDS=$(aws route53 list-resource-record-sets --hosted-zone-id $AWS_HOSTED_ZONE_ID --query "ResourceRecordSets[?Type == 'A'] | [?contains(Name, '$RECORD_NAME')]")

# for simplicity & avoiding need for jq,
# we assume aws does not return whitespace in the json array

if [ "$AAAA_RECORDS" == "[]" ]; then
  echo "AAAA record does not exist. Creating..."
  aws route53 change-resource-record-sets \
      --hosted-zone-id $AWS_HOSTED_ZONE_ID \
      --change-batch "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$RECORD_NAME\",\"Type\":\"AAAA\",\"TTL\":60,\"ResourceRecords\":[{\"Value\":\"$IPV6\"}]}}]}"
else
  echo "AAAA record exists. Not updating."
fi

if [ "$A_RECORDS" == "[]" ]; then
  echo "A record does not exist. Creating..."
  aws route53 change-resource-record-sets \
      --hosted-zone-id $AWS_HOSTED_ZONE_ID \
      --change-batch "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$RECORD_NAME\",\"Type\":\"A\",\"TTL\":60,\"ResourceRecords\":[{\"Value\":\"$IPV4\"}]}}]}"
else
  echo "A record exists. Not updating."
fi