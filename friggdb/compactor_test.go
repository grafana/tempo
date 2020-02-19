package friggdb

/*
func TestCompactor(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	_, _, err = New(&Config{
		Backend: "local",
		Local: &local.Config{
			Path: path.Join(tempDir, "traces"),
		},
		WALFilepath:              path.Join(tempDir, "wal"),
		IndexDownsample:          17,
		BloomFilterFalsePositive: .01,
		BlocklistRefreshRate:     30 * time.Minute,
	}, log.NewNopLogger())
	assert.NoError(t, err)

}*/

/*func createAndWriteBlock(w Writer) ([][]ID, [][]byte, uuid.UUID) {
	wal, err := w.WAL()
}*/
