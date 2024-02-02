package types

import (
	"errors"
	"fmt"
	"time"

	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/crypto/tmhash"
)

const (
	// MaxBlockSizeBytes is the maximum permitted size of the blocks.
	MaxBlockSizeBytes = 100 * 1024 * 1024

	// BlockPartSizeBytes is the size of one block part.
	BlockPartSizeBytes uint32 = 65536 // 64kB

	// MaxBlockPartsCount is the maximum number of block parts.
	MaxBlockPartsCount = (MaxBlockSizeBytes / BlockPartSizeBytes) + 1

	ABCIPubKeyTypeEd25519   = ed25519.KeyType
	ABCIPubKeyTypeSecp256k1 = secp256k1.KeyType
)

var ABCIPubKeyTypesToNames = map[string]string{
	ABCIPubKeyTypeEd25519:   ed25519.PubKeyName,
	ABCIPubKeyTypeSecp256k1: secp256k1.PubKeyName,
}

// ConsensusParams contains consensus critical parameters that determine the
// validity of blocks.
type ConsensusParams struct {
	Block     BlockParams     `json:"block"`
	Evidence  EvidenceParams  `json:"evidence"`
	Validator ValidatorParams `json:"validator"`
	Version   VersionParams   `json:"version"`
	ABCI      ABCIParams      `json:"abci"`
	Synchrony SynchronyParams `json:"synchrony"`
	PBTS      PBTSParams      `json:"pbts"`
}

// BlockParams define limits on the block size and gas plus minimum time
// between blocks.
type BlockParams struct {
	MaxBytes int64 `json:"max_bytes"`
	MaxGas   int64 `json:"max_gas"`
}

// EvidenceParams determine how we handle evidence of malfeasance.
type EvidenceParams struct {
	MaxAgeNumBlocks int64         `json:"max_age_num_blocks"` // only accept new evidence more recent than this
	MaxAgeDuration  time.Duration `json:"max_age_duration"`
	MaxBytes        int64         `json:"max_bytes"`
}

// ValidatorParams restrict the public key types validators can use.
// NOTE: uses ABCI pubkey naming, not Amino names.
type ValidatorParams struct {
	PubKeyTypes []string `json:"pub_key_types"`
}

type VersionParams struct {
	App uint64 `json:"app"`
}

// ABCIParams configure ABCI functionality specific to the Application Blockchain
// Interface.
type ABCIParams struct {
	VoteExtensionsEnableHeight int64 `json:"vote_extensions_enable_height"`
}

// VoteExtensionsEnabled returns true if vote extensions are enabled at height h
// and false otherwise.
func (a ABCIParams) VoteExtensionsEnabled(h int64) bool {
	if h < 1 {
		panic(fmt.Errorf("cannot check if vote extensions enabled for height %d (< 1)", h))
	}
	if a.VoteExtensionsEnableHeight == 0 {
		return false
	}
	return a.VoteExtensionsEnableHeight <= h
}

type PBTSParams struct {
	PBTSEnableHeight int64 `json:"pbts_enable_height"`
}

// PBTSEnabled returns true if PBTS are enabled at height h
// and false otherwise.
func (p PBTSParams) PBTSEnabled(h int64) bool {
	if h < 1 {
		panic(fmt.Errorf("cannot check if PBTS enabled for height %d (< 1)", h))
	}
	if p.PBTSEnableHeight == 0 {
		return false
	}
	return p.PBTSEnableHeight <= h
}

// SynchronyParams influence the validity of block timestamps.
// For more information on the relationship of the synchrony parameters to
// block validity, see the Proposer-Based Timestamps specification:
// https://github.com/tendermint/spec/blob/master/spec/consensus/proposer-based-timestamp/README.md
type SynchronyParams struct {
	Precision    time.Duration `json:"precision,string"`
	MessageDelay time.Duration `json:"message_delay,string"`
}

// DefaultConsensusParams returns a default ConsensusParams.
func DefaultConsensusParams() *ConsensusParams {
	return &ConsensusParams{
		Block:     DefaultBlockParams(),
		Evidence:  DefaultEvidenceParams(),
		Validator: DefaultValidatorParams(),
		Version:   DefaultVersionParams(),
		ABCI:      DefaultABCIParams(),
		Synchrony: DefaultSynchronyParams(),
		PBTS:      DefaultPBTSParams(),
	}
}

// DefaultBlockParams returns a default BlockParams.
func DefaultBlockParams() BlockParams {
	return BlockParams{
		MaxBytes: 4194304,  // four megabytes
		MaxGas:   10000000, // ten million
	}
}

// DefaultEvidenceParams returns a default EvidenceParams.
func DefaultEvidenceParams() EvidenceParams {
	return EvidenceParams{
		MaxAgeNumBlocks: 100000, // 27.8 hrs at 1block/s
		MaxAgeDuration:  48 * time.Hour,
		MaxBytes:        1048576, // 1MB
	}
}

// DefaultValidatorParams returns a default ValidatorParams, which allows
// only ed25519 pubkeys.
func DefaultValidatorParams() ValidatorParams {
	return ValidatorParams{
		PubKeyTypes: []string{ABCIPubKeyTypeEd25519},
	}
}

func DefaultVersionParams() VersionParams {
	return VersionParams{
		App: 0,
	}
}

func DefaultABCIParams() ABCIParams {
	return ABCIParams{
		// When set to 0, vote extensions are not required.
		VoteExtensionsEnableHeight: 0,
	}
}

func DefaultSynchronyParams() SynchronyParams {
	// TODO(@wbanfield): Determine experimental values for these defaults
	// https://github.com/tendermint/tendermint/issues/7202
	return SynchronyParams{
		Precision:    500 * time.Millisecond,
		MessageDelay: 2 * time.Second,
	}
}

// Disabled by default.
func DefaultPBTSParams() PBTSParams {
	return PBTSParams{
		PBTSEnableHeight: 0,
	}
}

func IsValidPubkeyType(params ValidatorParams, pubkeyType string) bool {
	for i := 0; i < len(params.PubKeyTypes); i++ {
		if params.PubKeyTypes[i] == pubkeyType {
			return true
		}
	}
	return false
}

// ValidateBasic validates the ConsensusParams to ensure all values are within their
// allowed limits, and returns an error if they are not.
func (params ConsensusParams) ValidateBasic() error {
	if params.Block.MaxBytes == 0 {
		return fmt.Errorf("block.MaxBytes cannot be 0")
	}
	if params.Block.MaxBytes < -1 {
		return fmt.Errorf("block.MaxBytes must be -1 or greater than 0. Got %d",

			params.Block.MaxBytes)
	}
	if params.Block.MaxBytes > MaxBlockSizeBytes {
		return fmt.Errorf("block.MaxBytes is too big. %d > %d",
			params.Block.MaxBytes, MaxBlockSizeBytes)
	}

	if params.Block.MaxGas < -1 {
		return fmt.Errorf("block.MaxGas must be greater or equal to -1. Got %d",
			params.Block.MaxGas)
	}

	if params.Evidence.MaxAgeNumBlocks <= 0 {
		return fmt.Errorf("evidence.MaxAgeNumBlocks must be greater than 0. Got %d",
			params.Evidence.MaxAgeNumBlocks)
	}

	if params.Evidence.MaxAgeDuration <= 0 {
		return fmt.Errorf("evidence.MaxAgeDuration must be greater than 0 if provided, Got %v",
			params.Evidence.MaxAgeDuration)
	}

	maxBytes := params.Block.MaxBytes
	if maxBytes == -1 {
		maxBytes = int64(MaxBlockSizeBytes)
	}
	if params.Evidence.MaxBytes > maxBytes {
		return fmt.Errorf("evidence.MaxBytesEvidence is greater than upper bound, %d > %d",
			params.Evidence.MaxBytes, params.Block.MaxBytes)
	}

	if params.Evidence.MaxBytes < 0 {
		return fmt.Errorf("evidence.MaxBytes must be non negative. Got: %d",
			params.Evidence.MaxBytes)
	}

	if params.ABCI.VoteExtensionsEnableHeight < 0 {
		return fmt.Errorf("ABCI.VoteExtensionsEnableHeight cannot be negative. Got: %d", params.ABCI.VoteExtensionsEnableHeight)
	}

	if params.Synchrony.MessageDelay <= 0 {
		return fmt.Errorf("synchrony.MessageDelay must be greater than 0. Got: %d",
			params.Synchrony.MessageDelay)
	}

	if params.Synchrony.Precision <= 0 {
		return fmt.Errorf("synchrony.Precision must be greater than 0. Got: %d",
			params.Synchrony.Precision)
	}

	if params.PBTS.PBTSEnableHeight < 0 {
		return fmt.Errorf("PBTS.PBTSEnableHeight must not be negative. Got: %d", params.PBTS.PBTSEnableHeight)
	}

	if len(params.Validator.PubKeyTypes) == 0 {
		return errors.New("len(Validator.PubKeyTypes) must be greater than 0")
	}

	// Check if keyType is a known ABCIPubKeyType
	for i := 0; i < len(params.Validator.PubKeyTypes); i++ {
		keyType := params.Validator.PubKeyTypes[i]
		if _, ok := ABCIPubKeyTypesToNames[keyType]; !ok {
			return fmt.Errorf("params.Validator.PubKeyTypes[%d], %s, is an unknown pubkey type",
				i, keyType)
		}
	}

	return nil
}

func (params ConsensusParams) ValidateUpdate(updated *cmtproto.ConsensusParams, h int64) error {
	var err error
	// Validate ABCI Update
	if updated.Abci != nil {
		if err = validateUpdateABCI(params, updated, h); err != nil {
			return err
		}
	}

	// Validate PBTS Update
	if updated.Pbts != nil {
		err = validateUpdatePBTS(params, updated, h)
	}
	return err
}

func validateUpdateABCI(params ConsensusParams, updated *cmtproto.ConsensusParams, h int64) error {
	if params.ABCI.VoteExtensionsEnableHeight == updated.Abci.VoteExtensionsEnableHeight {
		return nil
	}
	if params.ABCI.VoteExtensionsEnableHeight != 0 && updated.Abci.VoteExtensionsEnableHeight == 0 {
		return errors.New("vote extensions cannot be disabled once enabled")
	}
	if updated.Abci.VoteExtensionsEnableHeight <= h {
		return fmt.Errorf("VoteExtensionsEnableHeight cannot be updated to a past height, "+
			"initial height: %d, current height %d",
			params.ABCI.VoteExtensionsEnableHeight, h)
	}
	if params.ABCI.VoteExtensionsEnableHeight <= h {
		return fmt.Errorf("VoteExtensionsEnableHeight cannot be modified once"+
			"the initial height has occurred, "+
			"initial height: %d, current height %d",
			params.ABCI.VoteExtensionsEnableHeight, h)
	}
	return nil
}

func validateUpdatePBTS(params ConsensusParams, updated *cmtproto.ConsensusParams, h int64) error {
	if params.PBTS.PBTSEnableHeight != 0 {
		return errors.New("pbts already enabled")
	}

	if updated.Pbts.PbtsEnableHeight <= h {
		return fmt.Errorf("PbtsEnableHeight cannot be updated to a past height, "+
			"pbts enabled height: %d, current height %d",
			params.PBTS.PBTSEnableHeight, h)
	}
	return nil
}

// Hash returns a hash of a subset of the parameters to store in the block header.
// Only the Block.MaxBytes and Block.MaxGas are included in the hash.
// This allows the ConsensusParams to evolve more without breaking the block
// protocol. No need for a Merkle tree here, just a small struct to hash.
func (params ConsensusParams) Hash() []byte {
	hasher := tmhash.New()

	hp := cmtproto.HashedParams{
		BlockMaxBytes: params.Block.MaxBytes,
		BlockMaxGas:   params.Block.MaxGas,
	}

	bz, err := hp.Marshal()
	if err != nil {
		panic(err)
	}

	_, err = hasher.Write(bz)
	if err != nil {
		panic(err)
	}
	return hasher.Sum(nil)
}

// Update returns a copy of the params with updates from the non-zero fields of p2.
// NOTE: note: must not modify the original.
func (params ConsensusParams) Update(params2 *cmtproto.ConsensusParams) ConsensusParams {
	res := params // explicit copy

	if params2 == nil {
		return res
	}

	// we must defensively consider any structs may be nil
	if params2.Block != nil {
		res.Block.MaxBytes = params2.Block.MaxBytes
		res.Block.MaxGas = params2.Block.MaxGas
	}
	if params2.Evidence != nil {
		res.Evidence.MaxAgeNumBlocks = params2.Evidence.MaxAgeNumBlocks
		res.Evidence.MaxAgeDuration = params2.Evidence.MaxAgeDuration
		res.Evidence.MaxBytes = params2.Evidence.MaxBytes
	}
	if params2.Validator != nil {
		// Copy params2.Validator.PubkeyTypes, and set result's value to the copy.
		// This avoids having to initialize the slice to 0 values, and then write to it again.
		res.Validator.PubKeyTypes = append([]string{}, params2.Validator.PubKeyTypes...)
	}
	if params2.Version != nil {
		res.Version.App = params2.Version.App
	}
	if params2.Abci != nil {
		res.ABCI.VoteExtensionsEnableHeight = params2.Abci.GetVoteExtensionsEnableHeight()
	}
	if params2.Synchrony != nil {
		if params2.Synchrony.MessageDelay != nil {
			res.Synchrony.MessageDelay = *params2.Synchrony.GetMessageDelay()
		}
		if params2.Synchrony.Precision != nil {
			res.Synchrony.Precision = *params2.Synchrony.GetPrecision()
		}
	}
	if params2.Pbts != nil {
		res.PBTS.PBTSEnableHeight = params2.Pbts.GetPbtsEnableHeight()
	}
	return res
}

func (params *ConsensusParams) ToProto() cmtproto.ConsensusParams {
	return cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: params.Block.MaxBytes,
			MaxGas:   params.Block.MaxGas,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: params.Evidence.MaxAgeNumBlocks,
			MaxAgeDuration:  params.Evidence.MaxAgeDuration,
			MaxBytes:        params.Evidence.MaxBytes,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: params.Validator.PubKeyTypes,
		},
		Version: &cmtproto.VersionParams{
			App: params.Version.App,
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: params.ABCI.VoteExtensionsEnableHeight,
		},
		Synchrony: &cmtproto.SynchronyParams{
			MessageDelay: &params.Synchrony.MessageDelay,
			Precision:    &params.Synchrony.Precision,
		},
		Pbts: &cmtproto.PBTSParams{
			PbtsEnableHeight: params.PBTS.PBTSEnableHeight,
		},
	}
}

func ConsensusParamsFromProto(pbParams cmtproto.ConsensusParams) ConsensusParams {
	c := ConsensusParams{
		Block: BlockParams{
			MaxBytes: pbParams.Block.MaxBytes,
			MaxGas:   pbParams.Block.MaxGas,
		},
		Evidence: EvidenceParams{
			MaxAgeNumBlocks: pbParams.Evidence.MaxAgeNumBlocks,
			MaxAgeDuration:  pbParams.Evidence.MaxAgeDuration,
			MaxBytes:        pbParams.Evidence.MaxBytes,
		},
		Validator: ValidatorParams{
			PubKeyTypes: pbParams.Validator.PubKeyTypes,
		},
		Version: VersionParams{
			App: pbParams.Version.App,
		},
	}
	if pbParams.Abci != nil {
		c.ABCI.VoteExtensionsEnableHeight = pbParams.Abci.GetVoteExtensionsEnableHeight()
	}
	if pbParams.Synchrony != nil {
		if pbParams.Synchrony.MessageDelay != nil {
			c.Synchrony.MessageDelay = *pbParams.Synchrony.GetMessageDelay()
		}
		if pbParams.Synchrony.Precision != nil {
			c.Synchrony.Precision = *pbParams.Synchrony.GetPrecision()
		}
	}
	if pbParams.Pbts != nil {
		c.PBTS.PBTSEnableHeight = pbParams.Pbts.GetPbtsEnableHeight()
	}
	return c
}
