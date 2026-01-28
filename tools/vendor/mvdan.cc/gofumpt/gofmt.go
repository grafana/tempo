// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"

	"golang.org/x/sync/semaphore"

	gformat "mvdan.cc/gofumpt/format"
	"mvdan.cc/gofumpt/internal/govendor/diff"
	"mvdan.cc/gofumpt/internal/govendor/go/printer"
	gversion "mvdan.cc/gofumpt/internal/version"
)

//go:generate go run gen_govendor.go
//go:generate go run . -w internal/govendor

var (
	// main operation modes
	list      = flag.Bool("l", false, "")
	write     = flag.Bool("w", false, "")
	doDiff    = flag.Bool("d", false, "")
	allErrors = flag.Bool("e", false, "")

	// debugging
	cpuprofile = flag.String("cpuprofile", "", "")

	// gofumpt's own flags
	langVersion = flag.String("lang", "", "")
	modulePath  = flag.String("modpath", "", "")
	extraRules  = flag.Bool("extra", false, "")
	showVersion = flag.Bool("version", false, "")

	// DEPRECATED
	rewriteRule = flag.String("r", "", "")
	simplifyAST = flag.Bool("s", false, "")
)

var version = ""

// Keep these in sync with go/format/format.go.
const (
	tabWidth    = 8
	printerMode = printer.UseSpaces | printer.TabIndent | printerNormalizeNumbers

	// printerNormalizeNumbers means to canonicalize number literal prefixes
	// and exponents while printing. See https://golang.org/doc/go1.13#gofmt.
	//
	// This value is defined in go/printer specifically for go/format and cmd/gofmt.
	printerNormalizeNumbers = 1 << 30
)

// fdSem guards the number of concurrently-open file descriptors.
//
// For now, this is arbitrarily set to 200, based on the observation that many
// platforms default to a kernel limit of 256. Ideally, perhaps we should derive
// it from rlimit on platforms that support that system call.
//
// File descriptors opened from outside of this package are not tracked,
// so this limit may be approximate.
var fdSem = make(chan bool, 200)

var (
	fileSet    = token.NewFileSet() // per process FileSet
	parserMode parser.Mode
)

func usage() {
	fmt.Fprintf(os.Stderr, `usage: gofumpt [flags] [path ...]
	-version  show version and exit

	-d        display diffs instead of rewriting files
	-e        report all errors (not just the first 10 on different lines)
	-l        list files whose formatting differs from gofumpt's
	-w        write result to (source) file instead of stdout
	-extra    enable extra rules which should be vetted by a human

	-lang       str    target Go version in the form "go1.X" (default from go.mod)
	-modpath    str    Go module path containing the source file (default from go.mod)
`)
}

func initParserMode() {
	parserMode = parser.ParseComments | parser.SkipObjectResolution
	if *allErrors {
		parserMode |= parser.AllErrors
	}
}

func isGoFile(f fs.DirEntry) bool {
	// ignore non-Go files
	name := f.Name()
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !f.IsDir()
}

var rxCodeGenerated = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

func isGenerated(file *ast.File) bool {
	for _, cg := range file.Comments {
		if cg.Pos() > file.Package {
			return false
		}
		for _, line := range cg.List {
			if rxCodeGenerated.MatchString(line.Text) {
				return true
			}
		}
	}
	return false
}

// A sequencer performs concurrent tasks that may write output, but emits that
// output in a deterministic order.
type sequencer struct {
	maxWeight int64
	sem       *semaphore.Weighted   // weighted by input bytes (an approximate proxy for memory overhead)
	prev      <-chan *reporterState // 1-buffered
}

// newSequencer returns a sequencer that allows concurrent tasks up to maxWeight
// and writes tasks' output to out and err.
func newSequencer(maxWeight int64, out, err io.Writer) *sequencer {
	sem := semaphore.NewWeighted(maxWeight)
	prev := make(chan *reporterState, 1)
	prev <- &reporterState{out: out, err: err}
	return &sequencer{
		maxWeight: maxWeight,
		sem:       sem,
		prev:      prev,
	}
}

// exclusive is a weight that can be passed to a sequencer to cause
// a task to be executed without any other concurrent tasks.
const exclusive = -1

// Add blocks until the sequencer has enough weight to spare, then adds f as a
// task to be executed concurrently.
//
// If the weight is either negative or larger than the sequencer's maximum
// weight, Add blocks until all other tasks have completed, then the task
// executes exclusively (blocking all other calls to Add until it completes).
//
// f may run concurrently in a goroutine, but its output to the passed-in
// reporter will be sequential relative to the other tasks in the sequencer.
//
// If f invokes a method on the reporter, execution of that method may block
// until the previous task has finished. (To maximize concurrency, f should
// avoid invoking the reporter until it has finished any parallelizable work.)
//
// If f returns a non-nil error, that error will be reported after f's output
// (if any) and will cause a nonzero final exit code.
func (s *sequencer) Add(weight int64, f func(*reporter) error) {
	if weight < 0 || weight > s.maxWeight {
		weight = s.maxWeight
	}
	if err := s.sem.Acquire(context.TODO(), weight); err != nil {
		// Change the task from "execute f" to "report err".
		weight = 0
		f = func(*reporter) error { return err }
	}

	r := &reporter{prev: s.prev}
	next := make(chan *reporterState, 1)
	s.prev = next

	// Start f in parallel: it can run until it invokes a method on r, at which
	// point it will block until the previous task releases the output state.
	go func() {
		if err := f(r); err != nil {
			r.Report(err)
		}
		next <- r.getState() // Release the next task.
		s.sem.Release(weight)
	}()
}

// AddReport prints an error to s after the output of any previously-added
// tasks, causing the final exit code to be nonzero.
func (s *sequencer) AddReport(err error) {
	s.Add(0, func(*reporter) error { return err })
}

// GetExitCode waits for all previously-added tasks to complete, then returns an
// exit code for the sequence suitable for passing to os.Exit.
func (s *sequencer) GetExitCode() int {
	c := make(chan int, 1)
	s.Add(0, func(r *reporter) error {
		c <- r.ExitCode()
		return nil
	})
	return <-c
}

// A reporter reports output, warnings, and errors.
type reporter struct {
	prev  <-chan *reporterState
	state *reporterState
}

// reporterState carries the state of a reporter instance.
//
// Only one reporter at a time may have access to a reporterState.
type reporterState struct {
	out, err io.Writer
	exitCode int
}

// getState blocks until any prior reporters are finished with the reporter
// state, then returns the state for manipulation.
func (r *reporter) getState() *reporterState {
	if r.state == nil {
		r.state = <-r.prev
	}
	return r.state
}

// Warnf emits a warning message to the reporter's error stream,
// without changing its exit code.
func (r *reporter) Warnf(format string, args ...any) {
	fmt.Fprintf(r.getState().err, format, args...)
}

// Write emits a slice to the reporter's output stream.
//
// Any error is returned to the caller, and does not otherwise affect the
// reporter's exit code.
func (r *reporter) Write(p []byte) (int, error) {
	return r.getState().out.Write(p)
}

// Report emits a non-nil error to the reporter's error stream,
// changing its exit code to a nonzero value.
func (r *reporter) Report(err error) {
	if err == nil {
		panic("Report with nil error")
	}
	st := r.getState()
	switch err.(type) {
	case printedDiff:
		st.exitCode = 1
	default:
		scanner.PrintError(st.err, err)
		st.exitCode = 2
	}
}

func (r *reporter) ExitCode() int {
	return r.getState().exitCode
}

type printedDiff struct{}

func (printedDiff) Error() string { return "printed a diff, exiting with status code 1" }

// If info == nil, we are formatting stdin instead of a file.
// If in == nil, the source is the contents of the file with the given filename.
func processFile(filename string, info fs.FileInfo, in io.Reader, r *reporter, explicit bool) error {
	src, err := readFile(filename, info, in)
	if err != nil {
		return err
	}

	fileSet := token.NewFileSet()
	fragmentOk := false
	if info == nil {
		// If we are formatting stdin, we accept a program fragment in lieu of a
		// complete source file.
		fragmentOk = true
	}
	file, sourceAdj, indentAdj, err := parse(fileSet, filename, src, fragmentOk)
	if err != nil {
		return err
	}

	ast.SortImports(fileSet, file)

	// Apply gofumpt's changes before we print the code in gofumpt's format.

	// If either -lang or -modpath aren't set, fetch them from go.mod.
	lang := *langVersion
	modpath := *modulePath
	if lang == "" || modpath == "" {
		dir := filepath.Dir(filename)
		mod, ok := moduleCacheByDir.Load(dir)
		if ok && mod != nil {
			mod := mod.(*module)
			if mod.Go == "" {
				// If the go directive is missing, go 1.16 is assumed.
				// https://go.dev/ref/mod#go-mod-file-go
				mod.Go = "1.16"
			}
			if lang == "" {
				lang = "go" + mod.Go
			}
			if modpath == "" {
				modpath = mod.Module.Path
			}
		}
	}

	// We always apply the gofumpt formatting rules to explicit files, including stdin.
	// Otherwise, we don't apply them on generated files.
	// We also skip walking vendor directories entirely, but that happens elsewhere.
	if explicit || !isGenerated(file) {
		gformat.File(fileSet, file, gformat.Options{
			LangVersion: lang,
			ModulePath:  modpath,
			ExtraRules:  *extraRules,
		})
	}

	res, err := format(fileSet, file, sourceAdj, indentAdj, src, printer.Config{Mode: printerMode, Tabwidth: tabWidth})
	if err != nil {
		return err
	}

	if !bytes.Equal(src, res) {
		// formatting has changed
		if *list {
			fmt.Fprintln(r, filename)
		}
		if *write {
			if info == nil {
				panic("-w should not have been allowed with stdin")
			}
			// make a temporary backup before overwriting original
			perm := info.Mode().Perm()
			bakname, err := backupFile(filename+".", src, perm)
			if err != nil {
				return err
			}
			fdSem <- true
			err = os.WriteFile(filename, res, perm)
			<-fdSem
			if err != nil {
				os.Rename(bakname, filename)
				return err
			}
			err = os.Remove(bakname)
			if err != nil {
				return err
			}
		}
		if *doDiff {
			newName := filepath.ToSlash(filename)
			oldName := newName + ".orig"
			r.Write(diff.Diff(oldName, src, newName, res))
			return printedDiff{}
		}
	}

	if !*list && !*write && !*doDiff {
		_, err = r.Write(res)
	}

	return err
}

// readFile reads the contents of filename, described by info.
// If in is non-nil, readFile reads directly from it.
// Otherwise, readFile opens and reads the file itself,
// with the number of concurrently-open files limited by fdSem.
func readFile(filename string, info fs.FileInfo, in io.Reader) ([]byte, error) {
	if in == nil {
		fdSem <- true
		var err error
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		in = f
		defer func() {
			f.Close()
			<-fdSem
		}()
	}

	// Compute the file's size and read its contents with minimal allocations.
	//
	// If we have the FileInfo from filepath.WalkDir, use it to make
	// a buffer of the right size and avoid ReadAll's reallocations.
	//
	// If the size is unknown (or bogus, or overflows an int), fall back to
	// a size-independent ReadAll.
	size := -1
	if info != nil && info.Mode().IsRegular() && int64(int(info.Size())) == info.Size() {
		size = int(info.Size())
	}
	if size+1 <= 0 {
		// The file is not known to be regular, so we don't have a reliable size for it.
		var err error
		src, err := io.ReadAll(in)
		if err != nil {
			return nil, err
		}
		return src, nil
	}

	// We try to read size+1 bytes so that we can detect modifications: if we
	// read more than size bytes, then the file was modified concurrently.
	// (If that happens, we could, say, append to src to finish the read, or
	// proceed with a truncated buffer — but the fact that it changed at all
	// indicates a possible race with someone editing the file, so we prefer to
	// stop to avoid corrupting it.)
	src := make([]byte, size+1)
	n, err := io.ReadFull(in, src)
	switch err {
	case nil, io.EOF, io.ErrUnexpectedEOF:
		// io.ReadFull returns io.EOF (for an empty file) or io.ErrUnexpectedEOF
		// (for a non-empty file) if the file was changed unexpectedly. Continue
		// with comparing file sizes in those cases.
	default:
		return nil, err
	}
	if n < size {
		return nil, fmt.Errorf("error: size of %s changed during reading (from %d to %d bytes)", filename, size, n)
	} else if n > size {
		return nil, fmt.Errorf("error: size of %s changed during reading (from %d to >=%d bytes)", filename, size, len(src))
	}
	return src[:n], nil
}

func main() {
	// Arbitrarily limit in-flight work to 2MiB times the number of threads.
	//
	// The actual overhead for the parse tree and output will depend on the
	// specifics of the file, but this at least keeps the footprint of the process
	// roughly proportional to GOMAXPROCS.
	maxWeight := (2 << 20) * int64(runtime.GOMAXPROCS(0))
	s := newSequencer(maxWeight, os.Stdout, os.Stderr)

	// call gofmtMain in a separate function
	// so that it can use defer and have them
	// run before the exit.
	gofmtMain(s)
	os.Exit(s.GetExitCode())
}

func gofmtMain(s *sequencer) {
	// Ensure our parsed files never start with base 1,
	// to ensure that using token.NoPos+1 will panic.
	fileSet.AddFile("gofumpt_base.go", 1, 10)

	flag.Usage = usage
	flag.Parse()

	if *simplifyAST {
		fmt.Fprintf(os.Stderr, "warning: -s is deprecated as it is always enabled\n")
	}
	if *rewriteRule != "" {
		fmt.Fprintf(os.Stderr, `the rewrite flag is no longer available; use "gofmt -r" instead`+"\n")
		os.Exit(2)
	}

	// Print the gofumpt version if the user asks for it.
	if *showVersion {
		fmt.Println(gversion.String(version))
		return
	}

	if *cpuprofile != "" {
		fdSem <- true
		f, err := os.Create(*cpuprofile)
		if err != nil {
			s.AddReport(fmt.Errorf("creating cpu profile: %s", err))
			return
		}
		defer func() {
			f.Close()
			<-fdSem
		}()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	initParserMode()

	args := flag.Args()
	if len(args) == 0 {
		if *write {
			s.AddReport(fmt.Errorf("error: cannot use -w with standard input"))
			return
		}
		s.Add(0, func(r *reporter) error {
			// TODO: test explicit==true
			return processFile("<standard input>", nil, os.Stdin, r, true)
		})
		return
	}

	for _, arg := range args {
		switch info, err := os.Stat(arg); {
		case err != nil:
			s.AddReport(err)
		case !info.IsDir():
			// Non-directory arguments are always formatted.
			arg := arg
			s.Add(fileWeight(arg, info), func(r *reporter) error {
				return processFile(arg, info, nil, r, true)
			})
		default:
			// Directories are walked, ignoring non-Go files.
			err := filepath.WalkDir(arg, func(path string, f fs.DirEntry, err error) error {
				// vendor and testdata directories are skipped,
				// unless they are explicitly passed as an argument.
				base := filepath.Base(path)
				if path != arg && (base == "vendor" || base == "testdata") {
					return filepath.SkipDir
				}

				if err != nil || !isGoFile(f) {
					return err
				}
				info, err := f.Info()
				if err != nil {
					s.AddReport(err)
					return nil
				}
				s.Add(fileWeight(path, info), func(r *reporter) error {
					return processFile(path, info, nil, r, false)
				})
				return nil
			})
			if err != nil {
				s.AddReport(err)
			}
		}
	}
}

type module struct {
	Go     string
	Module struct {
		Path string
	}
}

func loadModuleInfo(dir string) any {
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Dir = dir

	// Spawning "go mod edit" will open files by design,
	// such as the named pipe to obtain stdout.
	// TODO(mvdan): if we run into "too many open files" errors again in the
	// future, we probably need to turn fdSem into a weighted semaphore so this
	// operation can acquire a weight larger than 1.
	fdSem <- true
	out, err := cmd.Output()
	defer func() { <-fdSem }()

	if err != nil || len(out) == 0 {
		return nil
	}
	mod := new(module)
	if err := json.Unmarshal(out, mod); err != nil {
		return nil
	}
	return mod
}

// Written to by fileWeight, read from fileWeight and processFile.
// A present but nil value means that loading the module info failed.
// Note that we don't require the keys to be absolute directories,
// so duplicates are possible. The same can happen with symlinks.
var moduleCacheByDir sync.Map // map[dirString]*module

func fileWeight(path string, info fs.FileInfo) int64 {
	dir := filepath.Dir(path)
	if _, ok := moduleCacheByDir.Load(dir); !ok {
		moduleCacheByDir.Store(dir, loadModuleInfo(dir))
	}
	if info == nil {
		return exclusive
	}
	if info.Mode().Type() == fs.ModeSymlink {
		var err error
		info, err = os.Stat(path)
		if err != nil {
			return exclusive
		}
	}
	if !info.Mode().IsRegular() {
		// For non-regular files, FileInfo.Size is system-dependent and thus not a
		// reliable indicator of weight.
		return exclusive
	}
	return info.Size()
}

const chmodSupported = runtime.GOOS != "windows"

// backupFile writes data to a new file named filename<number> with permissions perm,
// with <number randomly chosen such that the file name is unique. backupFile returns
// the chosen file name.
func backupFile(filename string, data []byte, perm fs.FileMode) (string, error) {
	fdSem <- true
	defer func() { <-fdSem }()

	// create backup file
	f, err := os.CreateTemp(filepath.Dir(filename), filepath.Base(filename))
	if err != nil {
		return "", err
	}
	bakname := f.Name()
	if chmodSupported {
		err = f.Chmod(perm)
		if err != nil {
			f.Close()
			os.Remove(bakname)
			return bakname, err
		}
	}

	// write data to backup file
	_, err = f.Write(data)
	if err1 := f.Close(); err == nil {
		err = err1
	}

	return bakname, err
}
