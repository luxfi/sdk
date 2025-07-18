// (c) 2023, Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

// WarpQuorumDenominator is the denominator used to calculate the
// quorum for Warp messages. A quorum is achieved when the sum of
// the validators' weights that signed the message is greater than
// (total_weight * quorum_numerator) / WarpQuorumDenominator.
const WarpQuorumDenominator = 3