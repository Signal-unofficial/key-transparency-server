/*
 * Copyright 2022 Signal Messenger, LLC
 * SPDX-License-Identifier: AGPL-3.0-only
 */
package org.signal.lambda;

import com.amazonaws.services.lambda.runtime.events.models.dynamodb.AttributeValue;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.google.common.annotations.VisibleForTesting;
import com.google.common.base.Preconditions;

import java.io.IOException;
import java.io.UncheckedIOException;
import java.nio.ByteBuffer;
import java.util.Arrays;
import java.util.Base64;
import java.util.HexFormat;
import java.util.Map;
import java.util.Objects;

record Account(
    String number,
    byte[] aci,
    byte[] aciIdentityKey,
    byte[] usernameHash) {

  @VisibleForTesting
  static final String KEY_ACCOUNT_UUID = "U";
  @VisibleForTesting
  static final String ATTR_ACCOUNT_E164 = "P";
  @VisibleForTesting
  static final String ATTR_ACCOUNT_USERNAME_HASH = "N";
  @VisibleForTesting
  static final String ATTR_ACCOUNT_DATA = "D";

  @Override
  public boolean equals(final Object o) {
    if (this == o)
      return true;
    if (o == null || getClass() != o.getClass())
      return false;
    Account account = (Account) o;
    return Objects.equals(number, account.number) &&
        Arrays.equals(aci, account.aci) &&
        Arrays.equals(aciIdentityKey, account.aciIdentityKey) &&
        Arrays.equals(usernameHash, account.usernameHash);
  }

  @Override
  public String toString() {
    return "Account{" +
        "number='" + number + '\'' +
        ", aci=" + Arrays.toString(aci) +
        ", aciIdentityKey=" + Arrays.toString(aciIdentityKey) +
        ", usernameHash=" + Arrays.toString(usernameHash) +
        '}';
  }

  public record Pair(Account prev, Account next) {

    /** Return a partition key used by Kinesis to group distributed updates.
     *
     * @return a string that Kinesis uses to group distributed updates.  If two Pairs have the same key,
     * their updates will go into the same kinesis shard, so their ordering will be maintained.  We simply
     * make the partition key reliant on the ACI, such that updates to the same account (and thus the same ACI)
     * are ordered.  Updates to different ACIs may go to different shards in the case where our Kinesis output
     * is sharded, and ordering across shards cannot be guaranteed.
     */
    public String partitionKey() {
      final byte[] aci = prev != null ? prev.aci : next.aci;
      return HexFormat.of().formatHex(aci, 0, 4);
    }
  }

  /** Private class used to parse the dynamodb 'D' field containing a base64 encoded JSON with account data in it. */
  private record AccountData(String identityKey) { }

  private static final ObjectMapper objectMapper = new ObjectMapper()
    .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES,false);

  static Account fromItem(Map<String, AttributeValue> item) {
    Preconditions.checkNotNull(item.get(KEY_ACCOUNT_UUID));
    Preconditions.checkNotNull(item.get(ATTR_ACCOUNT_E164));
    Preconditions.checkNotNull(item.get(ATTR_ACCOUNT_DATA));
    final byte[] uuid = new byte[16];
    item.get(KEY_ACCOUNT_UUID).getB().get(uuid);
    final ByteBuffer data = item.get(ATTR_ACCOUNT_DATA).getB().asReadOnlyBuffer();
    final byte[] identityKey;
    try {
      identityKey = Base64.getDecoder()
          .decode(objectMapper.readValue(new ByteBufferInputStream(data), AccountData.class).identityKey);
    } catch (IOException e) {
      throw new UncheckedIOException("IOException from reading bytes array", e);
    }
    final String number = item.get(ATTR_ACCOUNT_E164).getS();
    final byte[] usernameHash;
    final AttributeValue usernameHashAv = item.get(ATTR_ACCOUNT_USERNAME_HASH);
    if (usernameHashAv != null) {
      usernameHash = new byte[32];
      usernameHashAv.getB().asReadOnlyBuffer().get(usernameHash);
    } else {
      usernameHash = null;
    }
    return new Account(
        number,
        uuid,
        identityKey,
        usernameHash);
  }
}
