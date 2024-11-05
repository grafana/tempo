package gofakeit

import "math/rand"

// FileExtension will generate a random file extension
func FileExtension() string { return fileExtension(globalFaker.Rand) }

// FileExtension will generate a random file extension
func (f *Faker) FileExtension() string { return fileExtension(f.Rand) }

func fileExtension(r *rand.Rand) string { return getRandValue(r, []string{"file", "extension"}) }

// FileMimeType will generate a random mime file type
func FileMimeType() string { return fileMimeType(globalFaker.Rand) }

// FileMimeType will generate a random mime file type
func (f *Faker) FileMimeType() string { return fileMimeType(f.Rand) }

func fileMimeType(r *rand.Rand) string { return getRandValue(r, []string{"file", "mime_type"}) }

func addFileLookup() {
	AddFuncLookup("fileextension", Info{
		Display:     "File Extension",
		Category:    "file",
		Description: "Suffix appended to a filename indicating its format or type",
		Example:     "nes",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return fileExtension(r), nil
		},
	})

	AddFuncLookup("filemimetype", Info{
		Display:     "File Mime Type",
		Category:    "file",
		Description: "Defines file format and nature for browsers and email clients using standardized identifiers",
		Example:     "application/json",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return fileMimeType(r), nil
		},
	})
}
