# go-lru

This is an experiment of writing a simple tlru-like cache. Evictions are determined by a priority
queue ordered by expiration time. Each access resets the expiration time of an element in the cache.
