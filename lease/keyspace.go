package lease

/*
KeySpace normalizes resource keys and defines prefix coverage for exclusive leases.
*/
type KeySpace interface {
	Normalize(key string) (string, error)
	Covers(prefix, key string) bool
}
