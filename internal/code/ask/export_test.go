package ask

// TakeOver exposes the stale-lock takeover, whose losing arm needs a race
// between two runs that no black-box test can stage on demand.
var TakeOver = takeOver
