package domain

import "sync"

var (
    revMu       sync.RWMutex
    revokedByID = make(map[string]string) // id -> reason
)

func AddRevocation(id, reason string) {
    revMu.Lock()
    revokedByID[id] = reason
    revMu.Unlock()
}

func RemoveRevocation(id string) {
    revMu.Lock()
    delete(revokedByID, id)
    revMu.Unlock()
}

func IsRevoked(id string) (bool, string) {
    revMu.RLock()
    reason, ok := revokedByID[id]
    revMu.RUnlock()
    return ok, reason
}
