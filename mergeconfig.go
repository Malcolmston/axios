package axios

// MergeConfig deep-merges override onto base and returns the combined Config,
// mirroring axios.mergeConfig. Map-valued fields (headers, header groups,
// params) are merged key by key with override winning; scalar, pointer, slice
// and function fields are taken from override when set and otherwise from base.
//
// It is the same merge used internally by Create/Client.Create, exposed so
// callers can precompute a resolved configuration.
func MergeConfig(base, override Config) Config {
	return mergeConfig(base, override)
}
