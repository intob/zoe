#!/bin/bash

#
# Update AWS Route53 DNS records for lstn.swissinfo.ch
#

RECORD_NAME=$1

# Records must point to app's addresses on fly.io
while getopts ":ipv4:ipv6:aws-hosted-zone-id" opt; do
  case $opt in
    ipv4)
      IPV4_ADDRESS=$OPTARG
      ;;
    ipv6)
      IPV6_ADDRESS=$OPTARG
      ;;
    aws-hosted-zone-id)
      AWS_HOSTED_ZONE_ID=$OPTARG
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      ;;
  esac
done


# Check if records exist
AAAA_RECORDS=$(aws route53 list-resource-record-sets --hosted-zone-id $AWS_HOSTED_ZONE_ID --query "ResourceRecordSets[?Type == 'AAAA'] | [?contains(Name, '$RECORD_NAME')]")
A_RECORDS=$(aws route53 list-resource-record-sets --hosted-zone-id $AWS_HOSTED_ZONE_ID --query "ResourceRecordSets[?Type == 'A'] | [?contains(Name, '$RECORD_NAME')]")

# for simplicity & avoiding need for jq,
# we assume aws does not return whitespace in the json array

if [ "$AAAA_RECORDS" == "[]" ]; then
  echo "AAAA record does not exist. Creating..."
  aws route53 change-resource-record-sets \
      --hosted-zone-id $AWS_HOSTED_ZONE_ID \
      --change-batch "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$RECORD_NAME\",\"Type\":\"AAAA\",\"TTL\":60,\"ResourceRecords\":[{\"Value\":\"$IPV6_ADDRESS\"}]}}]}"
else
  echo "AAAA record exists. Not updating."
fi

if [ "$A_RECORDS" == "[]" ]; then
  echo "A record does not exist. Creating..."
  aws route53 change-resource-record-sets \
      --hosted-zone-id $AWS_HOSTED_ZONE_ID \
      --change-batch "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$RECORD_NAME\",\"Type\":\"A\",\"TTL\":60,\"ResourceRecords\":[{\"Value\":\"$IPV4_ADDRESS\"}]}}]}"
else
  echo "A record exists. Not updating."
fi