package tempofb

func (s *SearchPage) Contains(k []byte, v []byte, buffer *KeyValues) bool {
	return ContainsTag(s, buffer, k, v)
}
