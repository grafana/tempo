package main

import (
	"crypto/md5" //nolint:gosec // this is a test tool, md5 is used only for generating random strings
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
)

func main() {
	seed := flag.Int("seed", 0, "seed for the random number generator")
	num := flag.Int("n", 5000, "number of span names to generate")
	var features flagext.StringSlice
	flag.Var(&features, "features", "features to use in span names (comma separated list, can be repeated to add more types of spans)")

	flag.Parse()

	output := make([]string, *num)
	for i := 0; i < *num; i++ {
		output[i] = generateSpanName(*seed+i, features)
	}
	buf, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal output: %v", err)
	}
	fmt.Println(string(buf)) //nolint:forbidigo // this is a test tool
}

var templateRegex = regexp.MustCompile(`<[^>]+>`)

func generateSpanName(seed int, features []string) string {
	r := rand.New(rand.NewSource(int64(seed))) //nolint:gosec
	feature := features[r.Intn(len(features))]

	return templateRegex.ReplaceAllStringFunc(feature, func(match string) string {
		return generateFeature(match, r)
	})
}

func generateFeature(feature string, r *rand.Rand) string {
	switch feature {
	case "<ip>":
		return fmt.Sprintf("%d.%d.%d.%d", r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256))
	case "<base64>":
		return base64.StdEncoding.EncodeToString(randomBytes(r, 10))
	case "<uuid>":
		return uuid.New().String()
	case "<word>":
		return randomWord(r)
	case "<word-list>":
		return randomWordList(r)
	case "<method>":
		return randomMethod(r)
	case "<protocol>":
		return randomProtocol(r)
	case "<md5>":
		return hex.EncodeToString(md5.New().Sum(randomBytes(r, 10))) //nolint:gosec // this is a test tool
	case "<alphanumeric>":
		return randomAlphanumeric(r)
	case "<number>":
		return strconv.Itoa(r.Intn(1000))
	case "<sqlkeyword>":
		return randomSQLKeyword(r)
	case "<sql>":
		return randomSQL(r)
	default:
		return feature
	}
}

func randomSQLKeyword(r *rand.Rand) string {
	statements := []string{
		"SELECT",
		"INSERT",
		"UPDATE",
		"DELETE",
		"CREATE",
	}
	return statements[r.Intn(len(statements))]
}

func randomSQL(r *rand.Rand) string {
	keyword := randomSQLKeyword(r)

	switch keyword {
	case "SELECT":
		return "SELECT " + randomWordList(r) + " FROM " + randomWord(r)
	case "INSERT":
		return "INSERT INTO " + randomWord(r) + " (" + randomWordList(r) + ") VALUES (" + randomWordList(r) + ")"
	case "UPDATE":
		return "UPDATE " + randomWord(r) + " SET " + randomWordList(r) + " = " + randomWordList(r) + " WHERE " + randomWordList(r) + " = " + randomWordList(r)
	case "DELETE":
		return "DELETE FROM " + randomWord(r) + " WHERE " + randomWordList(r) + " = " + randomWordList(r)
	case "CREATE":
		return "CREATE TABLE " + randomWord(r) + " (" + randomWordList(r) + ")"
	default:
		return keyword
	}
}

func randomAlphanumeric(r *rand.Rand) string {
	length := r.Intn(10) + 1
	chars := make([]byte, length)
	for i := range chars {
		chars[i] = byte(r.Intn(10) + 48)
	}
	return string(chars)
}

func randomProtocol(r *rand.Rand) string {
	protocols := []string{
		"http",
		"https",
		"tcp",
		"udp",
	}
	return protocols[r.Intn(len(protocols))]
}

func randomBytes(r *rand.Rand, n int) []byte {
	buf := make([]byte, n)
	r.Read(buf)
	return buf
}

func randomMethod(r *rand.Rand) string {
	methods := []string{
		"GET",
		"POST",
		"PUT",
		"DELETE",
		"PATCH",
	}
	return methods[r.Intn(len(methods))]
}

func randomWordList(r *rand.Rand) string {
	length := r.Intn(10) + 1
	words := make([]string, length)
	for i := range words {
		words[i] = randomWord(r)
	}
	return strings.Join(words, "-")
}

func randomWord(r *rand.Rand) string {
	words := []string{
		"apple",
		"banana",
		"cherry",
		"date",
		"elderberry",
		"fig",
		"grape",
		"honeydew",
		"kiwi",
		"lemon",
		"lime",
		"mango",
		"nectarine",
		"orange",
		"pear",
		"pineapple",
	}
	return words[r.Intn(len(words))]
}
