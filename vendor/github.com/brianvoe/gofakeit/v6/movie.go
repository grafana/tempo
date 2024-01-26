package gofakeit

import "math/rand"

func MovieName() string { return movieName(globalFaker.Rand) }

func (f *Faker) MovieName() string { return movieName(f.Rand) }

func movieName(r *rand.Rand) string { return getRandValue(r, []string{"movie", "name"}) }

func MovieGenre() string { return movieGenre(globalFaker.Rand) }

func (f *Faker) MovieGenre() string { return movieGenre(f.Rand) }

func movieGenre(r *rand.Rand) string { return getRandValue(r, []string{"movie", "genre"}) }

type MovieInfo struct {
	Name  string `json:"name" xml:"name"`
	Genre string `json:"genre" xml:"genre"`
}

func Movie() *MovieInfo { return movie(globalFaker.Rand) }

func (f *Faker) Movie() *MovieInfo { return movie(f.Rand) }

func movie(r *rand.Rand) *MovieInfo {
	return &MovieInfo{
		Name:  movieName(r),
		Genre: movieGenre(r),
	}
}

func addMovieLookup() {
	AddFuncLookup("movie", Info{
		Display:     "Movie",
		Category:    "movie",
		Description: "Random Movie data set",
		Example:     `{name: "The Matrix", genre: "Action"}`,
		Output:      "map[string]string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return movie(r), nil
		},
	})

	AddFuncLookup("moviename", Info{
		Display:     "Movie Name",
		Category:    "movie",
		Description: "Random movie name",
		Example:     "The Matrix",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return movieName(r), nil
		},
	})

	AddFuncLookup("moviegenre", Info{
		Display:     "Genre",
		Category:    "movie",
		Description: "Random movie genre",
		Example:     "Action",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return movieGenre(r), nil
		},
	})
}
