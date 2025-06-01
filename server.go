package commit

import (
	"sync"
)

type StoreItem struct {
	value interface{}
	lock  sync.RWMutex
	// Any extra fields here
}

type Server struct {
	mu    sync.Mutex
	store map[string]*StoreItem
	// Your fields here
	txnLogs   map[int][]Operation      // list of the get or set operations done
	txnState  map[int]TransactionState // tid mapping to state
	heldLocks map[int][]string         // tid mapping to list of keys under a lock
}

// helper struct to simplify logged ops
type Operation struct {
	OpType string
	Key    string
	Value  interface{}
}

// Prepare handler
//
// This function should:
// 1. Attempt to obtain locks for the given transaction
// 2. If this succeeds, vote Yes
// 3. If this fails, release any obtained locks and vote No
func (sv *Server) Prepare(args *RPCArgs, reply *PrepareReply) {
	sv.mu.Lock()
	ops, ok := sv.txnLogs[args.Tid]
	curState := sv.txnState[args.Tid]
	sv.mu.Unlock()

	// check relevancy first
	if !ok {
		reply.Relevant = false
		reply.VoteYes = false
		return
	}
	// if no short circuit this is relevant
	reply.Relevant = true

	// already voted yes or is pre-committed or committed → just vote yes again
	if curState == stateVotedYes || curState == statePreCommitted || curState == stateCommitted {
		reply.VoteYes = true
		return
	}

	// already voted no or aborted → just vote no again
	if curState == stateVotedNo || curState == stateAborted {
		reply.VoteYes = false
		return
	}

	// lock all the keys used in this transaction
	lockedKeys := []string{}
	success := true

	for _, op := range ops {
		item, exists := sv.store[op.Key]
		if !exists {
			success = false
			break
		}
		if op.OpType == "get" {
			ok := item.lock.TryRLock()
			if !ok {
				success = false
				break
			}
		} else if op.OpType == "set" {
			ok := item.lock.TryLock()
			if !ok {
				success = false
				break
			}
		} else {
			continue
		}
		lockedKeys = append(lockedKeys, op.Key)
	}

	// if ANY can not be locked. release all
	if !success {
		for _, key := range lockedKeys {
			item := sv.store[key]
			// unlock according to operation type (lock type basically)
			for _, op := range ops {
				if op.Key == key {
					if op.OpType == "get" {
						item.lock.RUnlock()
					} else if op.OpType == "set" {
						item.lock.Unlock()
					}
				}
			}
		}

		sv.mu.Lock()
		sv.txnState[args.Tid] = stateVotedNo
		sv.mu.Unlock()

		reply.VoteYes = false
		return
	}

	// at this point this is a success, and we should record state and locks
	sv.mu.Lock()
	sv.txnState[args.Tid] = stateVotedYes
	sv.heldLocks[args.Tid] = lockedKeys
	sv.mu.Unlock()

	reply.VoteYes = true
}

// Abort handler
//
// This function should abort the given transaction
// Make sure to release any held locks
func (sv *Server) Abort(args *RPCArgs, reply *struct{}) {
	sv.mu.Lock()
	lockedKeys := sv.heldLocks[args.Tid]
	ops := sv.txnLogs[args.Tid] // grab ops once here
	delete(sv.heldLocks, args.Tid)
	sv.txnState[args.Tid] = stateAborted
	sv.mu.Unlock()

	for _, key := range lockedKeys {
		item := sv.store[key]

		for _, op := range ops {
			if op.Key != key {
				continue
			}
			if op.OpType == "get" {
				item.lock.RUnlock()
			} else if op.OpType == "set" {
				item.lock.Unlock()
			}
		}
	}
}

// Query handler
//
// This function should reply with information about all known transactions
func (sv *Server) Query(args struct{}, reply *QueryReply) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	reply.Transactions = make(map[int]TransactionState) // essentially a map copy to transactions field
	for tid, state := range sv.txnState {
		reply.Transactions[tid] = state
	}
}

// PreCommit handler
//
// This function should confirm that the server is ready to commit
func (sv *Server) PreCommit(args *RPCArgs, reply *struct{}) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.txnState[args.Tid] = statePreCommitted
}

// Commit handler
//
// This function should actually apply the logged operations
// Make sure to release any held locks
func (sv *Server) Commit(args *RPCArgs, reply *CommitReply) {
	reply.GetResults = make(map[string]interface{})

	sv.mu.Lock()
	ops := sv.txnLogs[args.Tid]
	curState := sv.txnState[args.Tid]
	lockedKeys := sv.heldLocks[args.Tid] // to make it more idempotent

	// if not already committed, apply writes and log the commits
	if curState != stateCommitted {
		for _, op := range ops {
			if op.OpType == "set" {
				sv.store[op.Key].value = op.Value
			}
		}
		sv.txnState[args.Tid] = stateCommitted
		delete(sv.heldLocks, args.Tid)
	}

	for _, op := range ops {
		if op.OpType == "get" {
			reply.GetResults[op.Key] = sv.store[op.Key].value
		}
	}
	sv.mu.Unlock()

	// we will then release any held locks only if we JUST committed
	if curState != stateCommitted {
		for _, key := range lockedKeys {
			item := sv.store[key]
			for _, op := range ops {
				if op.Key != key {
					continue
				}
				if op.OpType == "get" {
					item.lock.RUnlock()
				} else if op.OpType == "set" {
					item.lock.Unlock()
				}
			}
		}
	}
}

// Get
//
// This function should log a Get operation
func (sv *Server) Get(tid int, key string) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	// log the for op for this transaction
	sv.txnLogs[tid] = append(sv.txnLogs[tid], Operation{
		OpType: "get",
		Key:    key,
	})
}

// Set
//
// This function should log a Set operation
func (sv *Server) Set(tid int, key string, value interface{}) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	// log the set op for this transaction
	sv.txnLogs[tid] = append(sv.txnLogs[tid], Operation{
		OpType: "set", // same idea as above use the ds to maintain maps
		Key:    key,
		Value:  value,
	})
}

// Initialize new Server
// keys is a slice of the keys that this server is responsible for storing
func MakeServer(keys []string) *Server {
	store := make(map[string]*StoreItem)
	for _, key := range keys {
		store[key] = &StoreItem{} // each key will begin w/ an empty StoreItem
	}

	sv := &Server{
		store:     store,
		txnLogs:   make(map[int][]Operation),
		txnState:  make(map[int]TransactionState),
		heldLocks: make(map[int][]string),
	}

	return sv
}
