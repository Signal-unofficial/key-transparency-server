#!/bin/sh

# Not needed for LevelDB configuration
aws dynamodb create-table \
    --table-name "${KT_TABLE_NAME}" \
    --attribute-definitions AttributeName=k,AttributeType=S \
    --key-schema AttributeName=k,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST

aws dynamodb create-table \
    --table-name "${ACCOUNTS_TABLE_NAME}" \
    --attribute-definitions AttributeName=k,AttributeType=S \
    --key-schema AttributeName=k,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST
