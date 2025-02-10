package themis

//go:generate stringer -type=MpcAddrType
type MpcAddrType uint64

const (
	CommonMpcAddr       MpcAddrType = iota
	StateSubmitMpcAddr              // Special type for state submit
	RewardSubmitMpcAddr             // Special type for reward submit
	BlobSubmitMpcAddr               // Special type for blob submit
)

type MpcSignType int

const (
	BatchSubmitSignType MpcSignType = iota
	BatchRewardSignType
	CommitEpochToMetisSignType
	ReCommitEpochToMetisSignType
	L1UpdateMpcAddressSignType
	L2UpdateMpcAddressSignType
)
