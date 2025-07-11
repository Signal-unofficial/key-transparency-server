/*
 * Copyright 2022 Signal Messenger, LLC
 * SPDX-License-Identifier: AGPL-3.0-only
 */
package org.signal.lambda;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.amazonaws.services.lambda.runtime.events.DynamodbEvent;
import com.amazonaws.services.lambda.runtime.events.StreamsEventResponse;
import com.amazonaws.services.lambda.runtime.events.models.dynamodb.AttributeValue;
import com.amazonaws.services.lambda.runtime.events.models.dynamodb.StreamRecord;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.google.common.annotations.VisibleForTesting;
import software.amazon.awssdk.core.SdkBytes;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.kinesis.KinesisClient;
import software.amazon.awssdk.services.kinesis.model.PutRecordRequest;

import java.io.IOException;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Filters DynamoDb record updates for the subset relevant to key transparency, outputting them to Kinesis
 */
public class FilterKTUpdatesHandler implements RequestHandler<DynamodbEvent, Serializable> {

  private static final String KINESIS_OUTPUT_STREAM_ENVIRONMENT_VARIABLE = "KINESIS_OUTPUT_STREAM";
  private static final String KINESIS_OUTPUT_REGION_ENVIRONMENT_VARIABLE = "KINESIS_OUTPUT_REGION";

  @VisibleForTesting
  static final ObjectMapper OBJECT_MAPPER = new ObjectMapper()
      .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

  private final KinesisClient kinesisClient;
  private final String kinesisOutputStream;

  public FilterKTUpdatesHandler() {
    this(KinesisClient.builder()
            .region(Region.of(System.getenv(KINESIS_OUTPUT_REGION_ENVIRONMENT_VARIABLE)))
            .build(),
        System.getenv(KINESIS_OUTPUT_STREAM_ENVIRONMENT_VARIABLE));
  }

  @VisibleForTesting
  FilterKTUpdatesHandler(KinesisClient kinesisClient, String outputStream) {
    this.kinesisClient = kinesisClient;
    this.kinesisOutputStream = outputStream;
  }

  // https://docs.aws.amazon.com/lambda/latest/dg/with-ddb-create-package.html
  @Override
  public StreamsEventResponse handleRequest(final DynamodbEvent dbUpdate, final Context context) {
    List<StreamsEventResponse.BatchItemFailure> batchItemFailures = new ArrayList<>();
    String curRecordSequenceNumber = "";

    for (DynamodbEvent.DynamodbStreamRecord record : dbUpdate.getRecords()) {
      StreamRecord dbRecord = record.getDynamodb();
      curRecordSequenceNumber = dbRecord.getSequenceNumber();
      try {
        processRecord(dbRecord);
      } catch (Exception e) {
        batchItemFailures.add(new StreamsEventResponse.BatchItemFailure(curRecordSequenceNumber));
        e.printStackTrace();
      }
    }

    return new StreamsEventResponse(batchItemFailures);
  }

  // Modeled after https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/kds_gettingstarted.html
  @VisibleForTesting
  void processRecord(StreamRecord dbRecord) throws IOException {
    Account.Pair update = dbUpdateFor(dbRecord);
    if (update == null) return;
    kinesisClient.putRecord(PutRecordRequest
        .builder()
        .data(SdkBytes.fromByteArray(OBJECT_MAPPER.writeValueAsBytes(update)))
        .partitionKey(update.partitionKey())
        .streamName(kinesisOutputStream)
        .build());
  }

  private Account.Pair dbUpdateFor(StreamRecord dbRecord) {
    Map<String, AttributeValue> oldImage = dbRecord.getOldImage();
    Map<String, AttributeValue> newImage = dbRecord.getNewImage();
    if (oldImage == null || oldImage.isEmpty()) {
      return new Account.Pair(null, Account.fromItem(newImage));
    } else if (newImage == null || newImage.isEmpty()) {
      return new Account.Pair(Account.fromItem(oldImage), null);
    }
    Account oldAccount = Account.fromItem(oldImage);
    Account newAccount = Account.fromItem(newImage);
    if (oldAccount.equals(newAccount)) {
      return null;
    } else {
      return new Account.Pair(oldAccount, newAccount);
    }
  }
}
