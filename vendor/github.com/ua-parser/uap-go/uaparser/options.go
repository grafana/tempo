package uaparser

type Option func(*Parser)

func WithMode(mode LookupMode) Option {
	return func(s *Parser) {
		s.config.Mode = mode
	}
}

func WithSort(useSort bool, options ...SortOption) Option {
	return func(s *Parser) {
		s.config.UseSort = useSort

		for _, o := range options {
			o(s)
		}
	}
}

func WithDebug(debug bool) Option {
	return func(s *Parser) {
		s.config.DebugMode = debug
	}
}

func WithCacheSize(size int) Option {
	return func(s *Parser) {
		s.config.CacheSize = size
	}
}

func WithMatchIdxNotOk(idx int) Option {
	return func(s *Parser) {
		s.config.MatchIdxNotOk = idx
	}
}

func WithRegexDefinitions(def RegexDefinitions) Option {
	return func(s *Parser) {
		s.RegexDefinitions = &def
	}
}

type SortOption func(*Parser)

func WithMissesThreshold(threshold uint64) SortOption {
	return func(s *Parser) {
		s.config.MissesThreshold = threshold
	}
}
