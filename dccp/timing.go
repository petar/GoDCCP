// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

/*
   The Timestamp, Timestamp Echo, and Elapsed Time options help DCCP
   endpoints explicitly measure round-trip times.

13.1.  Timestamp Option

   This option is permitted in any DCCP packet.  The length of the
   option is 6 bytes.

   +--------+--------+--------+--------+--------+--------+
   |00101001|00000110|          Timestamp Value          |
   +--------+--------+--------+--------+--------+--------+
    Type=41  Length=6

   The four bytes of option data carry the timestamp of this packet.
   The timestamp is a 32-bit integer that increases monotonically with
   time, at a rate of 1 unit per 10 microseconds.  At this rate,
   Timestamp Value will wrap approximately every 11.9 hours.  Endpoints
   need not measure time at this fine granularity; for example, an
   endpoint that preferred to measure time at millisecond granularity
   might send Timestamp Values that were all multiples of 100.  The
   precise time corresponding to Timestamp Value zero is not specified:
   Timestamp Values are only meaningful relative to other Timestamp
   Values sent on the same connection.  A DCCP receiving a Timestamp
   option SHOULD respond with a Timestamp Echo option on the next packet
   it sends.

13.2.  Elapsed Time Option

   This option is permitted in any DCCP packet that contains an
   Acknowledgement Number; such options received on other packet types
   MUST be ignored.  It indicates how much time has elapsed since the
   packet being acknowledged -- the packet with the given
   Acknowledgement Number -- was received.  The option may take 4 or 6
   bytes, depending on the size of the Elapsed Time value.  Elapsed Time
   helps correct round-trip time estimates when the gap between
   receiving a packet and acknowledging that packet may be long -- in
   CCID 3, for example, where acknowledgements are sent infrequently.

   +--------+--------+--------+--------+
   |00101011|00000100|   Elapsed Time  |
   +--------+--------+--------+--------+
    Type=43    Len=4

   +--------+--------+--------+--------+--------+--------+
   |00101011|00000110|            Elapsed Time           |
   +--------+--------+--------+--------+--------+--------+
    Type=43    Len=6

   The option data, Elapsed Time, represents an estimated lower bound on
   the amount of time elapsed since the packet being acknowledged was
   received, with units of hundredths of milliseconds.  If Elapsed Time
   is less than a half-second, the first, smaller form of the option
   SHOULD be used.  Elapsed Times of more than 0.65535 seconds MUST be
   sent using the second form of the option.  The special Elapsed Time
   value 4294967295, which corresponds to approximately 11.9 hours, is
   used to represent any Elapsed Time greater than 42949.67294 seconds.
   DCCP endpoints MUST NOT report Elapsed Times that are significantly
   larger than the true elapsed times.  A connection MAY be reset with
   Reset Code 11, "Aggression Penalty", if one endpoint determines that
   the other is reporting a much-too-large Elapsed Time.

   Elapsed Time is measured in hundredths of milliseconds as a
   compromise between two conflicting goals.  First, it provides enough
   granularity to reduce rounding error when measuring elapsed time over
   fast LANs; second, it allows many reasonable elapsed times to fit
   into two bytes of data.

13.3.  Timestamp Echo Option

   This option is permitted in any DCCP packet, as long as at least one
   packet carrying the Timestamp option has been received.  Generally, a
   DCCP endpoint should send one Timestamp Echo option for each
   Timestamp option it receives, and it should send that option as soon
   as is convenient.  The length of the option is between 6 and 10
   bytes, depending on whether Elapsed Time is included and how large it
   is.

   +--------+--------+--------+--------+--------+--------+
   |00101010|00000110|           Timestamp Echo          |
   +--------+--------+--------+--------+--------+--------+
    Type=42    Len=6

   +--------+--------+------- ... -------+--------+--------+
   |00101010|00001000|  Timestamp Echo   |   Elapsed Time  |
   +--------+--------+------- ... -------+--------+--------+
    Type=42    Len=8       (4 bytes)

   +--------+--------+------- ... -------+------- ... -------+
   |00101010|00001010|  Timestamp Echo   |    Elapsed Time   |
   +--------+--------+------- ... -------+------- ... -------+
    Type=42   Len=10       (4 bytes)           (4 bytes)

   The first four bytes of option data, Timestamp Echo, carry a
   Timestamp Value taken from a preceding received Timestamp option.
   Usually, this will be the last packet that was received -- the packet
   indicated by the Acknowledgement Number, if any -- but it might be a
   preceding packet.  Each Timestamp received will generally result in
   exactly one Timestamp Echo transmitted.  If an endpoint has received
   multiple Timestamp options since the last time it sent a packet, then
   it MAY ignore all Timestamp options but the one included on the
   packet with the greatest sequence number.  Alternatively, it MAY
   include multiple Timestamp Echo options in its response, each
   corresponding to a different Timestamp option.

   The Elapsed Time value, similar to that in the Elapsed Time option,
   indicates the amount of time elapsed since receiving the packet whose
   timestamp is being echoed.  This time MUST have units of hundredths
   of milliseconds.  Elapsed Time is meant to help the Timestamp sender
   separate the network round-trip time from the Timestamp receiver's
   processing time.  This may be particularly important for CCIDs where
   acknowledgements are sent infrequently, so that there might be
   considerable delay between receiving a Timestamp option and sending
   the corresponding Timestamp Echo.  A missing Elapsed Time field is
   equivalent to an Elapsed Time of zero.  The smallest version of the
   option SHOULD be used that can hold the relevant Elapsed Time value.
*/
