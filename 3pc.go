package commit

// ------------------------------------------
//                  COMMON
// ------------------------------------------

type TransactionState int

const (
	stateOperations TransactionState = iota
	stateVotedNo
	stateVotedYes
	statePreCommitted
	stateAborted
	stateCommitted
)

// Common args struct because RPCs generally have the transaction ID as their only argument
type RPCArgs struct {
	Tid int
}

type PrepareReply struct {
	Relevant bool // essentially has this server logged any ops for the tid
	VoteYes  bool // if all locks are acquired toggle should be set to true, else false
}

type QueryReply struct {
	Transactions map[int]TransactionState // mapping tid to some transaction state
}

type CommitReply struct {
	GetResults map[string]interface{} // get the results for the client
}
